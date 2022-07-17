package models

// Item is an interface for any items for any vendors.
type Item interface {
	SellerSKU() string
	Stocks() int
}

// VendorClient is an interface for any vendor clients.
type VendorClient interface {
	TenantName() string
	Vendor() string
	CollectAllItems() ([]Item, error)
	LoadItem(sku string) (Item, error)
	SaveItem(item Item) error
}
