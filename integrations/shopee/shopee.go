// Package shopee implements interfacing with Shopee.
package shopee

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/nmcapule/oclz-go/integrations/models"
	"github.com/nmcapule/oclz-go/oauth2"
	"github.com/nmcapule/oclz-go/utils"
	"github.com/tidwall/gjson"

	log "github.com/sirupsen/logrus"
)

const Vendor = "SHOPEE"

// Config is a Lazada config.
type Config struct {
	Domain      string `json:"domain"`
	ShopID      int64  `json:"shop_id"`
	PartnerID   int64  `json:"partner_id"`
	PartnerKey  string `json:"partner_key"`
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

	var offset int64
	const limit = 50

	for {
		base, err := c.request(&http.Request{
			Method: http.MethodGet,
			URL: c.url("/api/v2/product/get_item_list", url.Values{
				"offset":      []string{strconv.FormatInt(offset, 10)},
				"page_size":   []string{strconv.FormatInt(limit, 10)},
				"item_status": []string{"NORMAL"},
			}),
		}, signatureMode(signatureModeShopAPI))
		if err != nil {
			return nil, fmt.Errorf("error response: %v", err)
		}

		for _, product := range base.Get("response.item").Array() {
			parsed, err := c.loadItemsFromProduct(int(product.Get("item_id").Int()))
			if err != nil {
				return nil, fmt.Errorf("load items from models: %v", err)
			}
			items = append(items, parsed...)
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
	base, err := c.request(&http.Request{
		Method: http.MethodGet,
		URL: c.url("/api/v2/product/search_item", url.Values{
			"page_size": []string{strconv.Itoa(10)},
			"item_sku":  []string{sku},
		}),
	}, signatureMode(signatureModeShopAPI))
	if err != nil {
		return nil, fmt.Errorf("error response: %v", err)
	}
	matches := base.Get("response.item_id_list").Array()
	if len(matches) == 0 {
		return nil, models.ErrNotFound
	}
	if len(matches) > 1 {
		log.Debugf("Multiple items retrieved for search_item %s: %+v", sku, matches)
	}
	// Iterate thru each initial search results.
	var items []*models.Item
	for _, match := range matches {
		parsed, err := c.loadItemsFromProduct(int(match.Int()))
		if err != nil {
			return nil, fmt.Errorf("load items: %v", err)
		}
		items = append(items, parsed...)
	}
	// Only include items that have exact SKU match from search.
	var filtered []*models.Item
	for i := range items {
		if items[i].SellerSKU == sku {
			filtered = append(filtered, items[i])
		}
	}
	items = filtered
	if len(items) == 0 {
		return nil, models.ErrNotFound
	}
	if len(items) > 1 {
		log.Warningf("Multiple items with same SKU retrieved for %s: %+v", sku, items)
		// return nil, models.ErrMultipleItems
	}
	return items[0], nil
}

// SaveItem saves item info for a single SKU.
// This only implements updating the product stock.
func (c *Client) SaveItem(item *models.Item) error {
	log.Warn("Cannot sync %q: SaveItem is unimplemented in %s", item.SellerSKU, c.Name)
	return nil
}

func (c *Client) loadItemsFromProduct(id int) ([]*models.Item, error) {
	base, err := c.request(&http.Request{
		Method: http.MethodGet,
		URL: c.url("/api/v2/product/get_item_base_info", url.Values{
			"item_id_list": []string{strconv.Itoa(id)},
		}),
	}, signatureMode(signatureModeShopAPI))
	if err != nil {
		return nil, fmt.Errorf("error response: %v", err)
	}

	var items []*models.Item
	for _, item := range base.Get("response.item_list").Array() {
		// If model exists, load from models endpoint instead.
		// This doesn't work unfortunately :/ No item has its `has_model == true`.
		if base.Get("response.has_model").Bool() {
			parsed, err := c.loadItemsFromModelOfItemID(id, item)
			if err != nil {
				return nil, fmt.Errorf("load from model: %v", err)
			}
			items = append(items, parsed...)
			continue
		}
		if item.Get("item_sku").String() == "" {
			log.Warningf("skipping item %d, empty sku", id)
			continue
		}
		items = append(items, &models.Item{
			SellerSKU: item.Get("item_sku").String(),
			Stocks:    int(item.Get("stock_info_v2.summary_info.total_available_stock").Int()),
			TenantProps: utils.GJSONFrom(map[string]any{
				"item_id":        id,
				"current_price":  item.Get("price_info.current_price").Float(),
				"original_price": item.Get("price_info.original_price").Float(),
				"currency":       item.Get("price_info.currency").String(),
				"item_name":      item.Get("item_name").String(),
			}),
		})
	}
	return items, nil
}

func (c *Client) loadItemsFromModelOfItemID(id int, item gjson.Result) ([]*models.Item, error) {
	base, err := c.request(&http.Request{
		Method: http.MethodGet,
		URL: c.url("/api/v2/product/get_model_list", url.Values{
			"item_id": []string{strconv.Itoa(id)},
		}),
	}, signatureMode(signatureModeShopAPI))
	if err != nil {
		return nil, fmt.Errorf("error response: %v", err)
	}

	var items []*models.Item
	for _, model := range base.Get("response.model").Array() {
		items = append(items, &models.Item{
			SellerSKU: model.Get("model_sku").String(),
			Stocks:    int(model.Get("stock_info_v2.summary_info.total_available_stock").Int()),
			TenantProps: utils.GJSONFrom(map[string]any{
				"item_id":        id,
				"model_id":       model.Get("model_id").Int(),
				"current_price":  model.Get("price_info.current_price").Float(),
				"original_price": model.Get("price_info.original_price").Float(),
				"currency":       model.Get("price_info.currency").String(),
				"item_name":      item.Get("item_name").String(),
			}),
		})
	}
	return items, nil
}
