package models

import (
	"time"

	"github.com/tidwall/gjson"

	pbm "github.com/pocketbase/pocketbase/models"
)

// Item is an interface for any items for any vendors.
type Item struct {
	ID          string
	TenantID    string
	SellerSKU   string
	Stocks      int
	TenantProps *gjson.Result
	Created     time.Time
	Updated     time.Time
}

// ItemFrom creates an item from a db record.
func ItemFrom(record *pbm.Record) *Item {
	tenantProps := gjson.Parse(record.GetString("tenant_props"))
	return &Item{
		ID:          record.GetId(),
		TenantID:    record.GetString("tenant"),
		SellerSKU:   record.GetString("seller_sku"),
		Stocks:      record.GetInt("stocks"),
		TenantProps: &tenantProps,
		Created:     record.GetTime("created"),
		Updated:     record.GetTime("updated"),
	}
}

// ToRecord converts an item into a db record.
func (i *Item) ToRecord(collection *pbm.Collection) *pbm.Record {
	record := pbm.NewRecord(collection)
	if i.ID != "" {
		record.MarkAsNotNew()
		record.Id = i.ID
	}
	record.Set("tenant", i.TenantID)
	record.Set("seller_sku", i.SellerSKU)
	record.Set("stocks", i.Stocks)
	record.Set("tenant_props", i.TenantProps.Raw)
	return record
}
