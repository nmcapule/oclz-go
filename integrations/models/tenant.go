package models

import (
	"encoding/json"

	"github.com/nmcapule/oclz-go/oauth2"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	pbm "github.com/pocketbase/pocketbase/models"
)

// Daemon is a background running service.
type Daemon interface {
	Start() error
}

// IntegrationClient is an interface for any vendor clients.
type IntegrationClient interface {
	Tenant() *BaseTenant
	CollectAllItems() ([]*Item, error)
	LoadItem(sku string) (*Item, error)
	SaveItem(item *Item) error
	CredentialsManager() oauth2.CredentialsManager
	Daemon() Daemon
}

type BaseTenant struct {
	ID          string
	Name        string
	Vendor      string
	Config      json.RawMessage
	TenantGroup string
}

func TenantFrom(record *pbm.Record) *BaseTenant {
	return &BaseTenant{
		ID:          record.GetId(),
		Name:        record.GetString("name"),
		Vendor:      record.GetString("vendor"),
		Config:      json.RawMessage(record.GetString("config")),
		TenantGroup: record.GetString("tenant_group"),
	}
}

func (b *BaseTenant) Tenant() *BaseTenant {
	return b
}

// BaseDatabaseTenant is a base tenant, with default implementations directly
// connected to the database.
type BaseDatabaseTenant struct {
	*BaseTenant
	Dao *daos.Dao
}

func (c *BaseDatabaseTenant) CollectAllItems() ([]*Item, error) {
	inventory, err := c.Dao.FindRecordsByExpr("tenant_inventory", dbx.HashExp{
		"tenant": c.ID,
	})
	if err != nil {
		return nil, err
	}
	var items []*Item
	for _, record := range inventory {
		items = append(items, ItemFrom(record))
	}
	return items, nil
}

func (c *BaseDatabaseTenant) LoadItem(sellerSKU string) (*Item, error) {
	inventory, err := c.Dao.FindRecordsByExpr("tenant_inventory", dbx.HashExp{
		"tenant":     c.ID,
		"seller_sku": sellerSKU,
	})
	if err != nil {
		return nil, err
	}
	if len(inventory) == 0 {
		return nil, ErrNotFound
	}
	if len(inventory) > 1 {
		return nil, ErrMultipleItems
	}
	return ItemFrom(inventory[0]), nil

}

func (c *BaseDatabaseTenant) SaveItem(item *Item) error {
	collection, err := c.Dao.FindCollectionByNameOrId("tenant_inventory")
	if err != nil {
		return err
	}
	return c.Dao.SaveRecord(item.ToRecord(collection))
}

func (c *BaseDatabaseTenant) CredentialsManager() oauth2.CredentialsManager {
	return nil
}

func (c *BaseDatabaseTenant) Daemon() Daemon {
	return nil
}
