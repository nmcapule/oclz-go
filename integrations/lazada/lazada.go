// Package lazada implements lazada tenant client.
package lazada

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/nmcapule/oclz-go/integrations/models"
	"github.com/nmcapule/oclz-go/oauth2"
	"github.com/tidwall/gjson"

	log "github.com/sirupsen/logrus"
)

const Vendor = "LAZADA"

func mustGJSON(v any) gjson.Result {
	b, err := json.Marshal(v)
	if err != nil {
		log.Fatalf("serializing JSON: %v", err)
	}
	return gjson.ParseBytes(b)
}

// Config is a Lazada config.
type Config struct {
	Domain      string `json:"domain"`
	AppKey      string `json:"app_key"`
	AppSecret   string `json:"app_secret"`
	RedirectURI string `json:"redirect_uri"`
}

// Client is a Lazada client.
type Client struct {
	*models.BaseTenant
	Config      *Config
	Credentials *oauth2.Credentials
}

// CollectAllItems collects and returns all items registered in this client.
func (c *Client) CollectAllItems() ([]*models.Item, error) {
	var items []*models.Item

	var offset int
	const limit = 50

	for {
		base, err := c.request(&http.Request{
			Method: http.MethodGet,
			URL: c.url("/products/get", url.Values{
				"offset": []string{strconv.Itoa(offset)},
				"limit":  []string{strconv.Itoa(limit)},
			}),
		})
		if err != nil {
			return nil, fmt.Errorf("send request: %v", err)
		}

		for _, product := range base.Get("data.products").Array() {
			items = append(items, parseItemsFromProduct(product)...)
		}

		log.WithFields(log.Fields{
			"items":  len(items),
			"offset": offset,
			"total":  base.Get("data.total_products").Int(),
		}).Infoln("loading items")

		offset += limit
		if offset >= int(base.Get("data.total_products").Int()) {
			break
		}
	}

	return items, nil
}

// LoadItem returns item info for a single SKU.
func (c *Client) LoadItem(sku string) (*models.Item, error) {
	base, err := c.request(&http.Request{
		Method: http.MethodGet,
		URL: c.url("/product/item/get", url.Values{
			"seller_sku": []string{sku},
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("send request: %v", err)
	}

	items := parseItemsFromProduct(base.Get("data"))
	if len(items) == 0 {
		return nil, models.ErrNotFound
	}
	// I'm not sure about this :P What if there are multiple items.
	return items[0], nil
}

// SaveItem saves item info for a single SKU.
// This only implements updating the product stock.
func (c *Client) SaveItem(item *models.Item) error {
	log.Warn("Cannot sync %q: SaveItem is unimplemented in %s", item.SellerSKU, c.Name)
	return nil
}

func parseItemsFromProduct(product gjson.Result) []*models.Item {
	var items []*models.Item
	for _, skuRaw := range product.Get("skus").Array() {
		sku := gjson.Parse(skuRaw.String())
		items = append(items, &models.Item{
			SellerSKU: sku.Get("SellerSku").String(),
			Stocks:    int(sku.Get("quantity").Int()),
			TenantProps: mustGJSON(map[string]interface{}{
				"item_id": product.Get("item_id").String(),
			}),
		})
	}
	return items
}
