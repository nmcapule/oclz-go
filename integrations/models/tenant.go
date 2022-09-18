package models

import (
	"encoding/json"

	"github.com/nmcapule/oclz-go/oauth2"
	pbm "github.com/pocketbase/pocketbase/models"
)

// VendorClient is an interface for any vendor clients.
type VendorClient interface {
	Tenant() *BaseTenant
	CollectAllItems() ([]*Item, error)
	LoadItem(sku string) (*Item, error)
	SaveItem(item *Item) error
	CredentialsManager() oauth2.CredentialsManager
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
		Name:        record.GetStringDataValue("name"),
		Vendor:      record.GetStringDataValue("vendor"),
		Config:      json.RawMessage(record.GetStringDataValue("config")),
		TenantGroup: record.GetStringDataValue("tenant_group"),
	}
}

func (b *BaseTenant) Tenant() *BaseTenant {
	return b
}
