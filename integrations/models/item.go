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
	tenantProps := gjson.Parse(record.GetStringDataValue("tenant_props"))
	return &Item{
		ID:          record.GetId(),
		TenantID:    record.GetStringDataValue("tenant"),
		SellerSKU:   record.GetStringDataValue("seller_sku"),
		Stocks:      record.GetIntDataValue("stocks"),
		TenantProps: &tenantProps,
		Created:     record.GetTimeDataValue("created"),
		Updated:     record.GetTimeDataValue("updated"),
	}
}

// ToRecord converts an item into a db record.
func (i *Item) ToRecord(collection *pbm.Collection) *pbm.Record {
	record := pbm.NewRecord(collection)
	if i.ID != "" {
		record.Id = i.ID
	}
	record.SetDataValue("tenant", i.TenantID)
	record.SetDataValue("seller_sku", i.SellerSKU)
	record.SetDataValue("stocks", i.Stocks)
	record.SetDataValue("tenant_props", i.TenantProps.Raw)
	return record
}
