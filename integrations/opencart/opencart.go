// Package opencart implements opencart tenant client.
package opencart

import (
	"net/http"

	"github.com/nmcapule/oclz-go/integrations/models"

	log "github.com/sirupsen/logrus"
)

const Vendor = "OPENCART"

// Config is a opencart config.
type Config struct {
	Domain   string `json:"domain"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Client is a opencart client.
type Client struct {
	*models.BaseTenant
	Config *Config
}

// CollectAllItems collects and returns all items registered in this client.
func (c *Client) CollectAllItems() ([]*models.Item, error) {
	base, err := c.request(&http.Request{
		Method: http.MethodGet,
		URL:    c.url("/module/store_sync/listlocalproducts", nil),
	})
	log.Fatalln("hello", base)
	return nil, err
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
