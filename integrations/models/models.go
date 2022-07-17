package models

import (
	"encoding/json"
	"errors"
	"time"

	pbm "github.com/pocketbase/pocketbase/models"
	"github.com/tidwall/gjson"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrMultipleItems = errors.New("unexpected multiple items retrieved")
	ErrUnimplemented = errors.New("not yet implemented")
)

// Item is an interface for any items for any vendors.
type Item struct {
	ID          string
	TenantID    string
	SellerSKU   string
	Stocks      int
	TenantProps gjson.Result
	Created     time.Time
	Updated     time.Time
}

func ItemFrom(record *pbm.Record) *Item {
	return &Item{
		ID:          record.GetId(),
		TenantID:    record.GetStringDataValue("tenant"),
		SellerSKU:   record.GetStringDataValue("seller_sku"),
		Stocks:      record.GetIntDataValue("stocks"),
		TenantProps: gjson.Parse(record.GetStringDataValue("tenant_props")),
		Created:     record.GetTimeDataValue("created"),
		Updated:     record.GetTimeDataValue("updated"),
	}
}

func (i *Item) ToRecord(collection *pbm.Collection) *pbm.Record {
	record := pbm.NewRecord(collection)
	if i.ID != "" {
		record.SetDataValue("id", i.ID)
	}
	record.SetDataValue("tenant", i.TenantID)
	record.SetDataValue("seller_sku", i.SellerSKU)
	record.SetDataValue("stocks", i.Stocks)
	record.SetDataValue("tenant_props", i.TenantProps.Raw)
	return record
}

// VendorClient is an interface for any vendor clients.
type VendorClient interface {
	Tenant() *BaseTenant
	CollectAllItems() ([]*Item, error)
	LoadItem(sku string) (*Item, error)
	SaveItem(item *Item) error
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
