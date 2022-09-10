// Package lazada implements lazada tenant client.
package lazada

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

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
	ShopID      string `json:"shop_id"`
	WarehouseID string `json:"warehouse_id"`
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
		res, err := http.DefaultClient.Do(c.prepare(&http.Request{
			Method: http.MethodGet,
			URL: c.url("/products/get", url.Values{
				"limit": []string{strconv.Itoa(limit)},
			}),
		}))
		if err != nil {
			return nil, fmt.Errorf("send request: %v", err)
		}
		defer res.Body.Close()

		b, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("parse body: %v", err)
		}
		base := gjson.ParseBytes(b)
		for _, product := range base.Get("data.products").Array() {
			items = append(items, parseItemsFromProduct(product)...)
		}

		offset += limit
		if offset >= int(base.Get("data.total_products").Int()) {
			break
		}
	}

	return items, nil
}

// LoadItem returns item info for a single SKU.
func (c *Client) LoadItem(sku string) (*models.Item, error) {
	res, err := http.DefaultClient.Do(c.prepare(&http.Request{
		Method: http.MethodGet,
		URL: c.url("/product/item/get", url.Values{
			"seller_sku": []string{sku},
		}),
	}))
	if err != nil {
		return nil, fmt.Errorf("send request: %v", err)
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("parse body: %v", err)
	}
	log.Infoln(string(b))
	items := parseItemsFromProduct(gjson.ParseBytes(b).Get("data"))
	if len(items) == 0 {
		return nil, models.ErrNotFound
	}
	// I'm not sure about this :P What if there are multiple items.
	return items[0], nil
}

// SaveItem saves item info for a single SKU.
// This only implements updating the product stock.
func (c *Client) SaveItem(item *models.Item) error {
	return nil
}

func (c *Client) url(endpoint string, query url.Values) *url.URL {
	u, err := url.Parse(c.Config.Domain + endpoint)
	if err != nil {
		log.WithFields(log.Fields{
			"domain":   c.Config.Domain,
			"endpoint": endpoint,
		}).Fatalln("Cannot parse URL")
	}
	return u
}

func (c *Client) prepare(req *http.Request) *http.Request {
	// Harvest endpoint and query from request.
	baseURL, _ := url.Parse(c.Config.Domain)
	endpoint := strings.TrimPrefix(req.URL.Path, baseURL.Path)
	query := req.URL.Query()
	query.Set("app_key", c.Config.AppKey)
	query.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	query.Set("access_token", c.Credentials.AccessToken)
	query.Set("sign_method", "sha256")
	query.Set("sign", signature(c.Config.AppSecret, endpoint, query))
	req.URL.RawQuery = query.Encode()
	return req
}

func signature(key, endpoint string, query url.Values) string {
	var sortedQuery []string
	for k := range query {
		sortedQuery = append(sortedQuery, fmt.Sprintf("%s%s", k, query.Get(k)))
	}
	sort.Strings(sortedQuery)
	base := fmt.Sprintf("%s%s", endpoint, strings.Join(sortedQuery, ""))
	h := hmac.New(sha256.New, []byte(key))
	if _, err := h.Write([]byte(base)); err != nil {
		log.Fatalf("encoding signature of %q: %v", base, err)
	}
	return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
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
