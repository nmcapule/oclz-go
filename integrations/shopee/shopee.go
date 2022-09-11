// Package shopee implements interfacing with Shopee.
package shopee

import (
	"github.com/nmcapule/oclz-go/integrations/models"
	"github.com/nmcapule/oclz-go/oauth2"
)

const Vendor = "SHOPEE"

// Config is a Lazada config.
type Config struct {
	Domain     string `json:"domain"`
	ShopID     int64  `json:"shop_id"`
	PartnerID  int64  `json:"partner_id"`
	PartnerKey string `json:"partner_key"`
}

// Client is a Lazada client.
type Client struct {
	*models.BaseTenant
	Config      *Config
	Credentials *oauth2.Credentials
}

// CollectAllItems collects and returns all items registered in this client.
func (c *Client) CollectAllItems() ([]*models.Item, error) {
	return nil, nil
}

// LoadItem returns item info for a single SKU.
func (c *Client) LoadItem(sku string) (*models.Item, error) {
	return nil, nil
}

// SaveItem saves item info for a single SKU.
// This only implements updating the product stock.
func (c *Client) SaveItem(item *models.Item) error {
	return nil
}
