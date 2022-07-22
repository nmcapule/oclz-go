package integrations

import (
	"encoding/json"
	"fmt"

	"github.com/nmcapule/oclz-go/integrations/intent"
	"github.com/nmcapule/oclz-go/integrations/models"
	"github.com/nmcapule/oclz-go/integrations/oauth2"
	"github.com/nmcapule/oclz-go/integrations/tiktok"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"

	log "github.com/sirupsen/logrus"
)

// LoadClient loads a client depending on the config vendor.
func LoadClient(dao *daos.Dao, tenantName string) (models.VendorClient, error) {
	collection, err := dao.FindCollectionByNameOrId("tenants")
	if err != nil {
		return nil, err
	}
	record, err := dao.FindFirstRecordByData(collection, "name", tenantName)
	if err != nil {
		return nil, err
	}

	oauth2Service := &oauth2.Service{Dao: dao}
	tenant := models.TenantFrom(record)
	switch tenant.Vendor {
	case intent.Vendor:
		var config intent.Config
		err := json.Unmarshal(tenant.Config, &config)
		if err != nil {
			return nil, err
		}
		return &intent.Client{
			BaseTenant: tenant,
			Dao:        dao,
		}, nil
	case tiktok.Vendor:
		var config tiktok.Config
		err := json.Unmarshal(tenant.Config, &config)
		if err != nil {
			return nil, err
		}
		credentials, err := oauth2Service.Load(tenant.ID)
		if err != nil {
			return nil, err
		}
		return &tiktok.Client{
			BaseTenant:  tenant,
			Config:      &config,
			Credentials: credentials,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported vendor %q", tenant.Vendor)
	}
}

// Syncer orchestrates how to sync items across multiple tenants.
type Syncer struct {
	TenantGroupName string
	Dao             *daos.Dao
	Tenants         map[string]models.VendorClient
	IntentTenant    models.VendorClient
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
		s.Tenants = make(map[string]models.VendorClient)
	}
	s.Tenants[tenantName] = tenant
	if tenant.Tenant().Vendor == intent.Vendor {
		s.IntentTenant = tenant
	}
	return nil
}

func (s *Syncer) NonIntentTenants() []models.VendorClient {
	var tenants []models.VendorClient
	for _, client := range s.Tenants {
		if client.Tenant().Vendor != intent.Vendor {
			tenants = append(tenants, client)
		}
	}
	return tenants
}

func (s *Syncer) TenantInventory(tenantName, sellerSKU string) (*models.Item, error) {
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

func (s *Syncer) SaveTenantInventory(tenantName string, item *models.Item) error {
	tenant := s.Tenants[tenantName]
	item.TenantID = tenant.Tenant().ID

	if tenantName == s.IntentTenant.Tenant().Name {
		return s.IntentTenant.SaveItem(item)
	}

	collection, err := s.Dao.FindCollectionByNameOrId("tenant_inventory")
	if err != nil {
		return err
	}
	return s.Dao.SaveRecord(item.ToRecord(collection))
}

func (s *Syncer) CollectAllItems() error {
	intentItems, err := s.IntentTenant.CollectAllItems()
	if err != nil {
		return err
	}
	intentItemsLookup := make(map[string]struct{})
	for _, item := range intentItems {
		intentItemsLookup[item.SellerSKU] = struct{}{}
	}

	var itemsOutsideIntent []*models.Item
	for _, tenant := range s.NonIntentTenants() {
		items, err := tenant.CollectAllItems()
		if err != nil {
			return err
		}
		for i, item := range items {
			if _, ok := intentItemsLookup[item.SellerSKU]; !ok {
				itemsOutsideIntent = append(itemsOutsideIntent, items[i])
			}
		}
	}

	for _, item := range itemsOutsideIntent {
		err := s.SaveTenantInventory(s.IntentTenant.Tenant().Name, item)
		if err != nil {
			return err
		}
	}

	return nil
}

// SyncItem tries to sync a single seller sku across all tenants.
func (s *Syncer) SyncItem(sellerSKU string) error {
	tenantItemMap := make(map[string]*models.Item)
	var totalDelta int
	for _, tenant := range s.Tenants {
		item, err := tenant.LoadItem(sellerSKU)
		if err != nil {
			return fmt.Errorf("loading live item %q from %s: %v", sellerSKU, tenant.Tenant().Name, err)
		}
		cached, err := s.TenantInventory(tenant.Tenant().Name, sellerSKU)
		if err != nil {
			return fmt.Errorf("loading cached item %q from %s: %v", sellerSKU, tenant.Tenant().Name, err)
		}
		totalDelta += item.Stocks - cached.Stocks
		tenantItemMap[tenant.Tenant().Name] = item
	}

	targetStocks := tenantItemMap[s.IntentTenant.Tenant().Name].Stocks
	targetStocks += totalDelta
	if targetStocks < 0 {
		log.Printf("warning: %s has negative stocks, setting to 0", sellerSKU)
	}

	for _, tenant := range s.Tenants {
		item := tenantItemMap[tenant.Tenant().Name]
		// Skip update if has the same stock as the intent tenant.
		if item.Stocks == targetStocks {
			continue
		}
		item.Stocks = targetStocks
		if err := tenant.SaveItem(item); err != nil {
			return fmt.Errorf("saving live item %q from %s: %v", sellerSKU, tenant.Tenant().Name, err)
		}
		if err := s.SaveTenantInventory(tenant.Tenant().Name, item); err != nil {
			return fmt.Errorf("saving cached item %q from %s: %v", sellerSKU, tenant.Tenant().Name, err)
		}
	}

	return nil
}
