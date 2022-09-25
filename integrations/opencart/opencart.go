// Package opencart implements opencart tenant client.
package opencart

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/nmcapule/oclz-go/integrations/models"
	"github.com/tidwall/gjson"

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
	return c.loadPaginatedItems(nil)
}

// LoadItem returns item info for a single SKU.
func (c *Client) LoadItem(sku string) (*models.Item, error) {
	items, err := c.loadPaginatedItems(url.Values{
		"filter_model": []string{sku},
	})
	if err != nil {
		return nil, fmt.Errorf("retrieving %q: %v", sku, err)
	}

	// Discard all items that are not exact match of SKU.
	var filtered []*models.Item
	for i := range items {
		if items[i].SellerSKU != sku {
			continue
		}
		filtered = append(filtered, items[i])
	}
	items = filtered

	if len(items) == 0 {
		return nil, fmt.Errorf("%q not found", sku)
	}
	if len(items) > 1 {
		return nil, fmt.Errorf("multiple results for sku %q", sku)
	}
	return items[0], nil
}

func (c *Client) loadPaginatedItems(query url.Values) ([]*models.Item, error) {
	if query == nil {
		query = make(url.Values)
	}

	page := 1
	var items []*models.Item
	for {
		query.Set("page", strconv.Itoa(page))
		base, err := c.request(&http.Request{
			Method: http.MethodGet,
			URL:    c.url("/catalog/product", query),
		}, responseParser(catalogProductParser))
		if err != nil {
			return nil, fmt.Errorf("request to /catalog/product: %v", err)
		}
		for _, item := range base.Get("data.rows").Array() {
			items = append(items, &models.Item{
				SellerSKU: item.Get("model").String(),
				Stocks:    int(item.Get("quantity").Int()),
				TenantProps: mustGJSON(map[string]interface{}{
					"product_name": item.Get("product_name").String(),
					"price":        item.Get("price").Float(),
					"status":       item.Get("status").String(),
				}),
			})
		}
		log.WithFields(log.Fields{
			"items":  len(items),
			"offset": base.Get("data.offset").Int(),
			"total":  base.Get("data.total").Int(),
		}).Infof("%s: loading items", c.Name)

		if page == int(base.Get("data.pages").Int()) {
			break
		}
		page += 1
	}
	return items, nil
}

// SaveItem saves item info for a single SKU.
// This only implements updating the product stock.
func (c *Client) SaveItem(item *models.Item) error {
	log.Warn("Cannot sync %q: SaveItem is unimplemented in %s", item.SellerSKU, c.Name)
	return nil
}

func mustGJSON(v any) gjson.Result {
	b, err := json.Marshal(v)
	if err != nil {
		log.Fatalf("serializing JSON: %v", err)
	}
	return gjson.ParseBytes(b)
}
