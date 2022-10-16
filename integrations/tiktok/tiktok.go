package tiktok

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nmcapule/oclz-go/integrations/models"
	"github.com/nmcapule/oclz-go/oauth2"
	"github.com/nmcapule/oclz-go/utils"
	"github.com/nmcapule/oclz-go/utils/scheduler"
	"github.com/tidwall/gjson"

	log "github.com/sirupsen/logrus"
)

// Vendor is key name for tiktok clients.
const Vendor = "TIKTOK"

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
				log.Debugln("Skipping sku_id:%s, empty seller_sku", sku.Get("id").String())
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
				TenantProps: utils.GJSONFrom(map[string]interface{}{
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
			"tenant": c.Name,
			"items":  len(items),
			"offset": page * limit,
			"total":  base.Get("data.total").Int(),
		}).Infof("Loading fresh items")

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
	_, err := c.request(&http.Request{
		Method: http.MethodPut,
		URL:    c.url("/api/products/stocks", nil),
		Body: io.NopCloser(strings.NewReader(utils.GJSONFrom(map[string]any{
			"product_id": item.TenantProps.Get("product_id").String(),
			"skus": []map[string]interface{}{
				{
					"id": item.TenantProps.Get("sku_id").String(),
					"stock_infos": []map[string]interface{}{
						{
							"available_stock": item.Stocks,
							"warehouse_id":    c.Config.WarehouseID,
						},
					},
				},
			},
		}).String())),
	})
	if err != nil {
		return fmt.Errorf("error response: %v", err)
	}

	// Poll until the update is confirmed propagated to Tiktok.
	return scheduler.Retry(func() bool {
		log.WithFields(log.Fields{
			"tenant":     c.Name,
			"seller_sku": item.SellerSKU,
		}).Debugln("Confirming item update...")
		live, err := c.LoadItem(item.SellerSKU)
		if err != nil {
			log.WithFields(log.Fields{
				"tenant":     c.Name,
				"seller_sku": item.SellerSKU,
			}).Errorf("Failed to confirm item update: %v", err)
			return false
		}
		return live.Stocks == item.Stocks
	}, scheduler.RetryConfig{
		RetryWait:       time.Second,
		RetryLimit:      10,
		BackoffMultiply: 2,
	})
}
