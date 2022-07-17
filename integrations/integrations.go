package integrations

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nmcapule/oclz-go/integrations/models"
	"github.com/nmcapule/oclz-go/integrations/oauth2"
	"github.com/nmcapule/oclz-go/integrations/tiktok"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
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

	tenant := record.GetId()
	vendor := record.GetStringDataValue("vendor")
	configRaw := record.GetStringDataValue("config")

	switch vendor {
	case tiktok.Vendor:
		var config tiktok.Config
		err := json.Unmarshal([]byte(configRaw), &config)
		if err != nil {
			return nil, err
		}
		credentials, err := oauth2Service.Load(tenant)
		if err != nil {
			return nil, err
		}

		return &tiktok.Client{
			Name:        tenantName,
			Config:      &config,
			Credentials: credentials,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported vendor %q", vendor)
	}
}

// Syncer orchestrates how to sync items across multiple tenants.
type Syncer struct {
	Dao             *daos.Dao
	Tenants         map[string]models.VendorClient
	TenantGroupName string
}

// NewSyncer creates a new syncer instance.
func NewSyncer(dao *daos.Dao, tenantGroupName string) (*Syncer, error) {
	s := &Syncer{
		Dao:             dao,
		TenantGroupName: tenantGroupName,
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
		"tenant_group": group.GetStringDataValue("id"),
	})
	if err != nil {
		return err
	}
	for _, record := range records {
		if err := s.register(record.GetStringDataValue("name")); err != nil {
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
	return nil
}

type baseItem struct {
	sellerSKU string
	stocks    int
}

func (b *baseItem) SellerSKU() string { return b.sellerSKU }
func (b *baseItem) Stocks() int       { return b.stocks }

func (s *Syncer) Inventory(tenantName, sellerSKU string) (models.Item, error) {
	collection, err := s.Dao.FindCollectionByNameOrId("inventory")
	if err != nil {
		return nil, err
	}
	record, err := s.Dao.FindFirstRecordByData(collection, "seller_sku", sellerSKU)
	if err != nil {
		return nil, err
	}
	return &baseItem{
		sellerSKU: record.GetStringDataValue("seller_sku"),
		stocks:    record.GetIntDataValue("stocks"),
	}, nil
}

func (s *Syncer) TenantInventory(tenantName, sellerSKU string) (models.Item, error) {
	collection, err := s.Dao.FindCollectionByNameOrId("inventory")
	if err != nil {
		return nil, err
	}
	record, err := s.Dao.FindFirstRecordByData(collection, "seller_sku", sellerSKU)
	if err != nil {
		return nil, err
	}
	log.Println(record)
	return nil, nil
}

// SyncItem tries to sync a single seller sku across all tenants.
func (s *Syncer) SyncItem(sellerSKU string) error {

	return nil
}
