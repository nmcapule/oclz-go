package syncer

import (
	"fmt"
	"strconv"
	"time"

	"github.com/nmcapule/oclz-go/integrations/intent"
	"github.com/nmcapule/oclz-go/integrations/models"
	"github.com/nmcapule/oclz-go/oauth2"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"

	log "github.com/sirupsen/logrus"
)

// Syncer orchestrates how to sync items across multiple tenants.
type Syncer struct {
	TenantGroupName string
	Dao             *daos.Dao
	Tenants         map[string]models.IntegrationClient
	IntentTenant    models.IntegrationClient
}

// NewSyncer creates a new syncer instance.
func NewSyncer(dao *daos.Dao, tenantGroupName string) (*Syncer, error) {
	s := &Syncer{
		TenantGroupName: tenantGroupName,
		Dao:             dao,
	}
	err := s.registerTenantGroup(tenantGroupName)
	return s, err
}

// registerTenantGroup registers all tenants under the given tenant group name.
func (s *Syncer) registerTenantGroup(tenantGroupName string) error {
	groups, err := s.Dao.FindCollectionByNameOrId("tenant_groups")
	if err != nil {
		return err
	}
	group, err := s.Dao.FindFirstRecordByData(groups, "name", tenantGroupName)
	if err != nil {
		return err
	}
	tenants, err := s.Dao.FindCollectionByNameOrId("tenants")
	if err != nil {
		return err
	}
	records, err := s.Dao.FindRecordsByExpr(tenants, dbx.HashExp{
		"tenant_group": group.GetId(),
	})
	if err != nil {
		return err
	}

	for _, tenant := range records {
		if !tenant.GetBoolDataValue("enable") {
			continue
		}
		if err := s.register(tenant.GetStringDataValue("name")); err != nil {
			return err
		}
	}
	return nil
}

// Registers a new vendor client using the given tenant name.
func (s *Syncer) register(tenantName string) error {
	tenant, err := LoadClient(s.Dao, tenantName)
	if err != nil {
		return err
	}
	if s.Tenants == nil {
		s.Tenants = make(map[string]models.IntegrationClient)
	}
	s.Tenants[tenantName] = tenant
	if tenant.Tenant().Vendor == intent.Vendor {
		s.IntentTenant = tenant
	}
	return nil
}

func (s *Syncer) RefreshCredentials() error {
	const expiryThreshold = time.Hour

	now := time.Now()
	oauth2Service := &oauth2.Service{Dao: s.Dao}
	for _, tenant := range s.nonIntentTenants() {
		cm := tenant.CredentialsManager()
		if cm == nil {
			log.Warningf("Skip credentials refresh for %s, no credentials manager", tenant.Tenant().Name)
			continue
		}

		// Only refresh credentials if credentials is about to expire.
		if cm.CredentialsExpiry().Sub(now) >= expiryThreshold {
			log.Infof("Skip credentials refresh for %s, not yet near expiry", tenant.Tenant().Name)
			continue
		}

		log.Infof("Refreshing credentials for %s", tenant.Tenant().Name)
		credentials, err := cm.RefreshCredentials()
		if err != nil {
			return fmt.Errorf("refreshing credentials for %s: %v", tenant.Tenant().Name, err)
		}
		err = oauth2Service.Save(credentials)
		if err != nil {
			return fmt.Errorf("save credentials for %s: %v", tenant.Tenant().Name, err)
		}
	}
	return nil
}

func (s *Syncer) nonIntentTenants() []models.IntegrationClient {
	var tenants []models.IntegrationClient
	for _, client := range s.Tenants {
		if client.Tenant().Vendor != intent.Vendor {
			tenants = append(tenants, client)
		}
	}
	return tenants
}

func (s *Syncer) tenantInventory(tenantName, sellerSKU string) (*models.Item, error) {
	collection, err := s.Dao.FindCollectionByNameOrId("tenant_inventory")
	if err != nil {
		return nil, err
	}
	inventory, err := s.Dao.FindRecordsByExpr(collection, dbx.HashExp{
		"tenant":     s.Tenants[tenantName].Tenant().ID,
		"seller_sku": sellerSKU,
	})
	if err != nil {
		return nil, err
	}
	if len(inventory) == 0 {
		return nil, models.ErrNotFound
	}
	if len(inventory) > 1 {
		return nil, models.ErrMultipleItems
	}
	return models.ItemFrom(inventory[0]), nil
}

func (s *Syncer) saveTenantInventory(tenantName string, item *models.Item) error {
	tenant := s.Tenants[tenantName]
	item.TenantID = tenant.Tenant().ID

	if tenantName == s.IntentTenant.Tenant().Name {
		return s.IntentTenant.SaveItem(item)
	}

	collection, err := s.Dao.FindCollectionByNameOrId("tenant_inventory")
	if err != nil {
		return err
	}
	record := item.ToRecord(collection)
	return s.Dao.SaveRecord(record)
}

// SyncItem tries to sync a single seller sku across all tenants.
func (s *Syncer) SyncItem(sellerSKU string) error {
	tenantItemMap := make(map[string]*models.Item)
	var totalDelta int
	for _, tenant := range s.Tenants {
		cached, err := s.tenantInventory(tenant.Tenant().Name, sellerSKU)
		if err == models.ErrNotFound {
			log.WithFields(log.Fields{
				"seller_sku": sellerSKU,
				"tenant":     tenant.Tenant().Name,
			}).Infoln("Item not found")
			continue
		}
		if err != nil {
			return fmt.Errorf("loading cached item %q from %s: %v", sellerSKU, tenant.Tenant().Name, err)
		}
		current, err := tenant.LoadItem(sellerSKU)
		if err != nil {
			return fmt.Errorf("loading live item %q from %s: %v", sellerSKU, tenant.Tenant().Name, err)
		}
		totalDelta += current.Stocks - cached.Stocks

		if totalDelta != 0 {
			err := s.saveInventoryDelta(&inventoryDelta{
				Inventory: cached.ID,
				Field:     "stocks",
				NValue:    float64(current.Stocks),
				SValue:    strconv.Itoa(current.Stocks),
			})
			if err != nil {
				return fmt.Errorf("saving inventory delta for %s (%s): %v", sellerSKU, cached.ID, err)
			}
		}

		current.ID = cached.ID
		current.Created = cached.Created
		tenantItemMap[tenant.Tenant().Name] = current
	}

	targetStocks := tenantItemMap[s.IntentTenant.Tenant().Name].Stocks
	targetStocks += totalDelta
	if targetStocks < 0 {
		log.Printf("warning: %s has negative stocks, setting to 0", sellerSKU)
	}

	for _, tenant := range s.Tenants {
		item, ok := tenantItemMap[tenant.Tenant().Name]
		if !ok {
			log.WithFields(log.Fields{
				"seller_sku": sellerSKU,
				"tenant":     tenant.Tenant().Name,
			}).Infoln("Skip item sync, does not exist in tenant")
			continue
		}
		// Skip update if has the same stock as the intent tenant.
		if item.Stocks == targetStocks {
			continue
		}
		item.Stocks = targetStocks
		if err := tenant.SaveItem(item); err != nil {
			return fmt.Errorf("saving live item %q from %s: %v", sellerSKU, tenant.Tenant().Name, err)
		}
		if err := s.saveTenantInventory(tenant.Tenant().Name, item); err != nil {
			return fmt.Errorf("saving cached item %q from %s: %v", sellerSKU, tenant.Tenant().Name, err)
		}
	}

	return nil
}
