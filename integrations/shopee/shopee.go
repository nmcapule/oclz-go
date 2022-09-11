// Package shopee implements interfacing with Shopee.
package shopee

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
	var items []*models.Item

	var offset int64
	const limit = 50

	for {
		base, err := c.request(&http.Request{
			Method: http.MethodPost,
			URL: c.url("/api/v2/product/get_item_list", url.Values{
				"offset":      []string{strconv.FormatInt(offset, 10)},
				"page_size":   []string{strconv.FormatInt(limit, 10)},
				"item_status": []string{"NORMAL"},
			}),
		}, tokenRetrievalMode)
		if err != nil {
			return nil, fmt.Errorf("error response: %v", err)
		}

		for _, product := range base.Get("response.item").Array() {
			items = append(items, &models.Item{
				TenantProps: mustGJSON(map[string]interface{}{
					"item_id": product.Get("item_id").String(),
				}),
			})
		}

		log.WithFields(log.Fields{
			"items":  len(items),
			"offset": offset,
			"total":  base.Get("response.total_count").Int(),
		}).Infoln("loading items")

		if !base.Get("response.has_next_page").Bool() {
			break
		}
		offset = base.Get("response.next_offset").Int()
	}

	return items, nil
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

func mustGJSON(v any) gjson.Result {
	b, err := json.Marshal(v)
	if err != nil {
		log.Fatalf("serializing JSON: %v", err)
	}
	return gjson.ParseBytes(b)
}