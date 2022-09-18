package tiktok

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/nmcapule/oclz-go/integrations/models"
	"github.com/nmcapule/oclz-go/oauth2"
	"github.com/tidwall/gjson"

	log "github.com/sirupsen/logrus"
)

// Vendor is key name for tiktok clients.
const Vendor = "TIKTOK"

func mustGJSON(v any) gjson.Result {
	b, err := json.Marshal(v)
	if err != nil {
		log.Fatalf("serializing JSON: %v", err)
	}
	return gjson.ParseBytes(b)
}

// Config is a tiktok config.
type Config struct {
	Domain      string `json:"domain"`
	AppKey      string `json:"app_key"`
	AppSecret   string `json:"app_secret"`
	ShopID      string `json:"shop_id"`
	WarehouseID string `json:"warehouse_id"`
	RedirectURI string `json:"redirect_uri"`
}

// Client is a tiktok client.
type Client struct {
	*models.BaseTenant
	Config      *Config
	Credentials *oauth2.Credentials
}

func (c *Client) parseItemsFromSearch(data gjson.Result) []*models.Item {
	var items []*models.Item
	data.Get("products").ForEach(func(_, product gjson.Result) bool {
		product.Get("skus").ForEach(func(_, sku gjson.Result) bool {
			if sku.Get("seller_sku").String() == "" {
				log.Warningf("Skipping sku_id:%s, empty seller_sku", sku.Get("id").String())
				return true
			}

			stocks := 0
			sku.Get("stock_infos").ForEach(func(_, info gjson.Result) bool {
				if info.Get("warehouse_id").String() == c.Config.WarehouseID {
					stocks += int(info.Get("available_stock").Int())
				}
				return true
			})
			items = append(items, &models.Item{
				SellerSKU: sku.Get("seller_sku").String(),
				Stocks:    stocks,
				TenantProps: mustGJSON(map[string]interface{}{
					"product_id": product.Get("id").String(),
					"sku_id":     sku.Get("id").String(),
				}),
			})
			return true
		})
		return true
	})
	return items
}

func (c *Client) defaultWarehouseID() (string, error) {
	base, err := c.request(&http.Request{
		Method: http.MethodPost,
		URL:    c.url("/api/v2/product/get_item_list", url.Values{}),
	})
	return base.Get("data.warehouse_list.#(warehouse_type==1).warehouse_id").String(), err
}

// CollectAllItems collects and returns all items registered in this client.
func (c *Client) CollectAllItems() ([]*models.Item, error) {
	if c.Config.WarehouseID == "" {
		id, err := c.defaultWarehouseID()
		if err != nil {
			return nil, fmt.Errorf("retrieve warehouse: %v", err)
		}
		c.Config.WarehouseID = id
	}

	var items []*models.Item
	var page int64 = 1
	const limit = int64(50)
	for {
		base, err := c.request(&http.Request{
			Method: http.MethodPost,
			URL: c.url("/api/products/search", url.Values{
				"page_number": []string{strconv.FormatInt(page, 10)},
				"page_size":   []string{strconv.FormatInt(limit, 10)},
			}),
		})
		if err != nil {
			return nil, fmt.Errorf("error response: %v", err)
		}

		items = append(items, c.parseItemsFromSearch(base.Get("data"))...)

		log.WithFields(log.Fields{
			"items":  len(items),
			"offset": page * limit,
			"total":  base.Get("data.total").Int(),
		}).Infoln("loading items")

		if page*limit >= base.Get("data.total").Int() {
			break
		}
		page += 1
	}

	return items, nil
}

// LoadItem returns item info for a single SKU.
func (c *Client) LoadItem(sku string) (*models.Item, error) {
	body, err := json.Marshal(map[string]interface{}{
		"seller_sku_list": sku,
	})
	if err != nil {
		return nil, fmt.Errorf("compose payload: %v", err)
	}

	base, err := c.request(&http.Request{
		Method: http.MethodPost,
		URL: c.url("/api/products/search", url.Values{
			"page_number": []string{strconv.FormatInt(1, 10)},
			"page_size":   []string{strconv.FormatInt(50, 10)},
		}),
		Body: io.NopCloser(strings.NewReader(string(body))),
	})
	if err != nil {
		return nil, fmt.Errorf("error response: %v", err)
	}
	// Collect only items with matching SKU.
	items := c.parseItemsFromSearch(base.Get("data"))
	var filtered []*models.Item
	for i := range items {
		if items[i].SellerSKU == sku {
			filtered = append(filtered, items[i])
		}
	}
	items = filtered
	// Check if multiple or none matched.
	if len(items) == 0 {
		return nil, models.ErrNotFound
	}
	if len(items) > 1 {
		log.Warn("multiple items found for %q: %v", sku, models.ErrMultipleItems)
	}
	return items[0], nil
}

// SaveItem saves item info for a single SKU.
// This only implements updating the product stock.
func (c *Client) SaveItem(item *models.Item) error {
	log.Warn("Cannot sync %q: SaveItem is unimplemented in %s", item.SellerSKU, c.Name)
	// request := map[string]interface{}{
	// 	"product_id": item.TenantProps.Get("product_id").String(),
	// 	"skus": []map[string]interface{}{
	// 		{
	// 			"id": item.TenantProps.Get("sku_id").String(),
	// 			"stock_infos": []map[string]interface{}{
	// 				{
	// 					"available_stock": item.Stocks,
	// 					"warehouse_id":    c.Config.WarehouseID,
	// 				},
	// 			},
	// 		},
	// 	},
	// }
	// _, err := c.put("/api/products/stocks", request, nil)
	// return err
	return nil
}
