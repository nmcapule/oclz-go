// Package opencart implements opencart tenant client.
package opencart

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/nmcapule/oclz-go/integrations/models"
	"github.com/nmcapule/oclz-go/utils"
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
	return c.loadCatalogProductPages(nil)
}

// LoadItem returns item info for a single SKU.
func (c *Client) LoadItem(sku string) (*models.Item, error) {
	items, err := c.loadCatalogProductPages(url.Values{
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

// SaveItem saves item info for a single SKU.
// This only implements updating the product stock.
func (c *Client) SaveItem(item *models.Item) error {
	log.Warn("Cannot sync %q: SaveItem is unimplemented in %s", item.SellerSKU, c.Name)
	return models.ErrUnimplemented
}

func (c *Client) loadCatalogProductPages(query url.Values) ([]*models.Item, error) {
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
		}, responseParser(scrapeCatalogProduct))
		if err != nil {
			return nil, fmt.Errorf("request to /catalog/product: %v", err)
		}
		for _, row := range base.Get("data.rows").Array() {
			items = append(items, &models.Item{
				SellerSKU: row.Get("model").String(),
				Stocks:    int(row.Get("quantity").Int()),
				TenantProps: utils.GJSONFrom(map[string]interface{}{
					"product_name": row.Get("product_name").String(),
					"price":        row.Get("price").Float(),
					"status":       row.Get("status").String(),
				}),
			})
		}
		log.WithFields(log.Fields{
			"tenant": c.Name,
			"items":  len(items),
			"offset": base.Get("data.offset").Int(),
			"total":  base.Get("data.total").Int(),
		}).Infof("Loading fresh items")

		if page == int(base.Get("data.pages").Int()) {
			break
		}
		page += 1
	}
	return items, nil
}

func (c *Client) loadSaleOrderPages(query url.Values) (*gjson.Result, error) {
	if query == nil {
		query = make(url.Values)
	}

	page := 1
	var orders []map[string]any
	for {
		query.Set("page", strconv.Itoa(page))
		base, err := c.request(&http.Request{
			Method: http.MethodGet,
			URL:    c.url("/sale/order", query),
		}, responseParser(scrapeSaleOrder))
		if err != nil {
			return nil, fmt.Errorf("request to /sale/order: %v", err)
		}
		for _, row := range base.Get("data.rows").Array() {
			orders = append(orders, map[string]any{
				"seller_sku": row.Get("model").String(),
				"stocks":     row.Get("quantity").Float(),
			})
		}
		log.WithFields(log.Fields{
			"tenant": c.Name,
			"items":  len(orders),
			"offset": base.Get("data.offset").Int(),
			"total":  base.Get("data.total").Int(),
		}).Infof("Loading sale orders")

		if page == int(base.Get("data.pages").Int()) {
			break
		}
		page += 1
	}
	return utils.GJSONFrom(orders), nil
}
