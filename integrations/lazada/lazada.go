// Package lazada implements lazada tenant client.
package lazada

import (
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

const Vendor = "LAZADA"

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
			"tenant": c.Name,
			"items":  len(items),
			"offset": offset,
			"total":  base.Get("data.total_products").Int(),
		}).Infof("Loading fresh items")

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
	// Compose the payload.
	xml := fmt.Sprintf(`
		<Request>
			<Product>
				<Skus>
					<Sku>
						<ItemId>%d</ItemId>
						<SkuId>%d</SkuId>
						<SellerSku>%s</SellerSku>
						<Quantity>%d</Quantity>
					</Sku>
				</Skus>
			</Product>
		</Request>`,
		item.TenantProps.Get("item_id").Int(),
		item.TenantProps.Get("sku_id").Int(),
		item.SellerSKU,
		// TODO(nmcapule): Find a better way to resolve this. This depend on
		// the syncer always giving the latest state of the item.
		item.Stocks+int(item.TenantProps.Get("reserved").Int()))

	// Do the actual update.
	_, err := c.request(&http.Request{
		Method: http.MethodPost,
		URL:    c.url("/product/price_quantity/update", nil),
		Body: io.NopCloser(strings.NewReader(url.Values{
			"payload": []string{xml},
		}.Encode())),
	})
	if err != nil {
		return fmt.Errorf("send request: %v", err)
	}

	// Poll until the update is confirmed propagated to Lazada.
	return scheduler.Retry(func() bool {
		log.WithFields(log.Fields{
			"tenant":     c.Tenant().Name,
			"seller_sku": item.SellerSKU,
		}).Debugln("Confirming item update...")
		live, err := c.LoadItem(item.SellerSKU)
		if err != nil {
			log.WithFields(log.Fields{
				"tenant":     c.Tenant().Name,
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

func parseItemsFromProduct(product gjson.Result) []*models.Item {
	var items []*models.Item
	for _, skuRaw := range product.Get("skus").Array() {
		sku := gjson.Parse(skuRaw.String())

		var totalQuantity int64
		for _, n := range sku.Get("multiWarehouseInventories.#.totalQuantity").Array() {
			totalQuantity += n.Int()
		}
		var sellableQuantity = sku.Get("quantity").Int()

		items = append(items, &models.Item{
			SellerSKU: sku.Get("SellerSku").String(),
			Stocks:    int(sellableQuantity),
			TenantProps: utils.GJSONFrom(map[string]interface{}{
				"item_id":  product.Get("item_id").Int(),
				"sku_id":   sku.Get("SkuId").Int(),
				"shop_sku": sku.Get("ShopSku").String(),
				"price":    sku.Get("price").Float(),
				"reserved": totalQuantity - sellableQuantity,
			}),
		})
	}
	return items
}
