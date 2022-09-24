// Package opencart implements opencart tenant client.
package opencart

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
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
	base, err := c.request(&http.Request{
		Method: http.MethodGet,
		// URL:    c.url("/module/store_sync/listlocalproducts", nil),
		URL: c.url("/catalog/product", nil),
	}, responseParser(func(input string) string {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(input))
		if err != nil {
			log.Fatalf("parsing doc: %v", err)
		}
		var rows []map[string]interface{}
		doc.Find("#form-product > div > table > tbody > tr").Each(func(_ int, s *goquery.Selection) {
			rows = append(rows, map[string]interface{}{
				"model":        s.Children().Get(3).FirstChild.Data,
				"quantity":     s.Find("td:nth-child(5) > span").Text(),
				"product_name": s.Children().Get(2).FirstChild.Data,
				"price":        s.Children().Get(4).FirstChild.Data,
				"status":       s.Children().Get(6).FirstChild.Data,
			})
		})
		b, err := json.Marshal(rows)
		if err != nil {
			log.Fatalf("encoding rows: %v", err)
		}
		return string(b)
	}))
	var items []*models.Item
	base.ForEach(func(_, item gjson.Result) bool {
		items = append(items, &models.Item{
			SellerSKU: item.Get("model").String(),
			Stocks:    int(item.Get("quantity").Int()),
		})
		log.Infoln(&models.Item{
			SellerSKU: item.Get("model").String(),
			Stocks:    int(item.Get("quantity").Int()),
		})
		return true
	})
	log.Fatalln(items)
	return items, err
}

// LoadItem returns item info for a single SKU.
func (c *Client) LoadItem(sku string) (*models.Item, error) {
	log.Warn("Cannot load %q: LoadItem is unimplemented in %s", sku, c.Name)
	return nil, nil
}

// SaveItem saves item info for a single SKU.
// This only implements updating the product stock.
func (c *Client) SaveItem(item *models.Item) error {
	log.Warn("Cannot sync %q: SaveItem is unimplemented in %s", item.SellerSKU, c.Name)
	return nil
}
