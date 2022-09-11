package tiktok

import (
	"encoding/json"

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

type response struct {
	Code      int             `json:"code"`
	Data      json.RawMessage `json:"data"`
	Message   string          `json:"message"`
	RequestID string          `json:"request_id"`
}

// Config is a tiktok config.
type Config struct {
	Domain      string `json:"domain"`
	AppKey      string `json:"app_key"`
	AppSecret   string `json:"app_secret"`
	ShopID      string `json:"shop_id"`
	WarehouseID string `json:"warehouse_id"`
}

// Client is a tiktok client.
type Client struct {
	*models.BaseTenant
	Config      *Config
	Credentials *oauth2.Credentials
}

func (c *Client) parseItemsFromSearch(data json.RawMessage) []*models.Item {
	var items []*models.Item
	gjson.ParseBytes(data).Get("products").ForEach(func(_, product gjson.Result) bool {
		product.Get("skus").ForEach(func(_, sku gjson.Result) bool {
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

// CollectAllItems collects and returns all items registered in this client.
func (c *Client) CollectAllItems() ([]*models.Item, error) {
	if c.Config.WarehouseID == "" {
		res, err := c.get("/api/logistics/get_warehouse_list", nil)
		if err != nil {
			return nil, err
		}
		c.Config.WarehouseID = gjson.GetBytes(res.Data, "warehouse_list.#(warehouse_type==1).warehouse_id").String()
	}

	var items []*models.Item
	page := 1
	pageSize := 50
	for {
		payload := map[string]interface{}{
			"page_number": page,
			"page_size":   pageSize,
		}
		res, err := c.post("/api/products/search", payload, nil)
		if err != nil {
			return nil, err
		}
		items = append(items, c.parseItemsFromSearch(res.Data)...)
		total := gjson.ParseBytes(res.Data).Get("total").Int()
		if page*pageSize > int(total) {
			break
		}
		page += 1
	}

	return items, nil
}

// LoadItem returns item info for a single SKU.
func (c *Client) LoadItem(sku string) (*models.Item, error) {
	payload := map[string]interface{}{
		"page_number": 1,
		"page_size":   50,
	}
	res, err := c.post("/api/products/search", payload, nil)
	if err != nil {
		return nil, err
	}
	items := c.parseItemsFromSearch(res.Data)
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
	request := map[string]interface{}{
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
	}
	_, err := c.put("/api/products/stocks", request, nil)
	return err
}
