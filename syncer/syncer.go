package syncer

import (
	"fmt"
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
	Config          Config
	Logger          *log.Logger
}

// NewSyncer creates a new syncer instance.
func NewSyncer(dao *daos.Dao, tenantGroupName string) (*Syncer, error) {
	// Setup logger from standard logger. Note that this affects **all** logrus
	// loggers within the application.
	// TODO(nmcapule): Inject to every service that needs to log.
	logger := log.StandardLogger()
	logger.SetReportCaller(true)
	logger.AddHook(&LogHook{
		Dao: dao,
		LogLevels: []log.Level{
			log.InfoLevel,
			log.WarnLevel,
			log.ErrorLevel,
			log.FatalLevel,
			log.PanicLevel,
		},
	})

	s := &Syncer{
		TenantGroupName: tenantGroupName,
		Dao:             dao,
		Logger:          logger,
	}
	err := s.registerTenantGroup(tenantGroupName)
	if err != nil {
		return nil, fmt.Errorf("register tenant group: %v", err)
	}
	return s, nil
}

// registerTenantGroup registers all tenants under the given tenant group name.
func (s *Syncer) registerTenantGroup(tenantGroupName string) error {
	// Load tenant gruop.
	group, err := s.Dao.FindFirstRecordByData("tenant_groups", "name", tenantGroupName)
	if err != nil {
		return err
	}
	// Set config from tenant group.
	if err := s.loadConfigFromGroup(group); err != nil {
		return fmt.Errorf("loading tenant group config: %v", err)
	}

	// Load tenants.
	records, err := s.Dao.FindRecordsByExpr("tenants", dbx.HashExp{
		"tenant_group": group.GetId(),
	})
	if err != nil {
		return err
	}

	for _, tenant := range records {
		if !tenant.GetBool("enable") {
			continue
		}
		if err := s.register(tenant.GetString("name")); err != nil {
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
	const expiryThreshold = 6 * time.Hour

	now := time.Now()
	oauth2Service := &oauth2.Service{Dao: s.Dao}
	for _, tenant := range s.nonIntentTenants() {
		cm := tenant.CredentialsManager()
		if cm == nil {
			log.WithFields(log.Fields{
				"tenant": tenant.Tenant().Name,
			}).Debugln("Skip credentials refresh, no credentials manager")
			continue
		}

		// Only refresh credentials if credentials is about to expire.
		if cm.CredentialsExpiry().Sub(now) >= expiryThreshold {
			log.WithFields(log.Fields{
				"tenant": tenant.Tenant().Name,
			}).Debugln("Skip credentials refresh, not yet near expiry")
			continue
		}

		log.WithFields(log.Fields{
			"tenant": tenant.Tenant().Name,
		}).Debugln("Refreshing credentials")
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
	inventory, err := s.Dao.FindRecordsByExpr("tenant_inventory", dbx.HashExp{
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
	// Check if the record already exists in the collection.
	records, err := s.Dao.FindRecordsByExpr(collection.Name, dbx.HashExp{
		"seller_sku": item.SellerSKU,
		"tenant":     item.TenantID,
	})
	if err != nil {
		return fmt.Errorf("check if already exists: %v", err)
	}
	if len(records) > 0 {
		log.WithFields(log.Fields{
			"tenant":     tenantName,
			"seller_sku": item.SellerSKU,
		}).Debugln("Item already exists! Updating instead...")
		item.ID = records[0].GetId()
	}
	return s.Dao.SaveRecord(item.ToRecord(collection))
}
