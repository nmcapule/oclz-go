package tiktok

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/nmcapule/oclz-go/integrations/models"
	"github.com/nmcapule/oclz-go/integrations/oauth2"
	"github.com/tidwall/gjson"
)

// Vendor is key name for tiktok clients.
const Vendor = "TIKTOK"

var errUnimplemented = errors.New("not yet implemented")

type response struct {
	Code      int             `json:"code"`
	Data      json.RawMessage `json:"data"`
	Message   string          `json:"message"`
	RequestID string          `json:"request_id"`
}

type Item struct {
	productID string
	skuID     string
	sellerSKU string
	stocks    int
}

func (i *Item) Stocks() int {
	return i.stocks
}

func (i *Item) SellerSKU() string {
	return i.sellerSKU
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
	Name        string
	Config      *Config
	Credentials *oauth2.Credentials
}

// Vendor returns the tenant name.
func (c *Client) TenantName() string {
	return c.Name
}

// Vendor returns the vendor name.
func (c *Client) Vendor() string {
	return Vendor
}

// CollectAllItems collects and returns all items registered in this client.
func (c *Client) CollectAllItems() ([]models.Item, error) {
	if c.Config.WarehouseID == "" {
		res, err := c.get("/api/logistics/get_warehouse_list", nil)
		if err != nil {
			return nil, err
		}
		c.Config.WarehouseID = gjson.GetBytes(res.Data, "warehouse_list.#(warehouse_type==1).warehouse_id").String()
	}

	var items []models.Item
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
		data := gjson.ParseBytes(res.Data)
		data.Get("products").ForEach(func(_, product gjson.Result) bool {
			product.Get("skus").ForEach(func(_, sku gjson.Result) bool {
				stocks := 0
				sku.Get("stock_infos").ForEach(func(_, info gjson.Result) bool {
					stocks += int(info.Get("available_stock").Int())
					return true
				})
				items = append(items, &Item{
					productID: product.Get("id").String(),
					skuID:     sku.Get("id").String(),
					sellerSKU: sku.Get("seller_sku").String(),
					stocks:    stocks,
				})
				return true
			})
			return true
		})

		total := data.Get("total").Int()
		if page*pageSize > int(total) {
			break
		}
		page += 1
	}

	return items, nil
}

// LoadItem returns item info for a single SKU.
func (c *Client) LoadItem(sku string) (models.Item, error) {
	return nil, errUnimplemented
}

// SaveItem saves item info for a single SKU.
func (c *Client) SaveItem(item models.Item) error {
	return errUnimplemented
}

func (c *Client) signature(endpoint string, query url.Values) string {
	var queryConcats []string
	for k := range query {
		queryConcats = append(queryConcats, fmt.Sprintf("%s%s", k, query.Get(k)))
	}
	sort.Strings(queryConcats)

	base := fmt.Sprintf(
		"%s%s%s%s",
		c.Config.AppSecret,
		endpoint,
		strings.Join(queryConcats, ""),
		c.Config.AppSecret,
	)
	h := hmac.New(sha256.New, []byte(c.Config.AppSecret))
	if _, err := h.Write([]byte(base)); err != nil {
		log.Fatalf("encoding signature of %q: %v", base, err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (c *Client) prepareURL(endpoint string, query url.Values) string {
	if query == nil {
		query = make(url.Values)
	}
	query.Set("timestamp", fmt.Sprintf("%d", time.Now().Unix()))
	query.Set("app_key", c.Config.AppKey)
	query.Set("shop_id", c.Config.ShopID)
	query.Set("sign", c.signature(endpoint, query))
	query.Set("access_token", c.Credentials.AccessToken)

	return fmt.Sprintf("%s%s?%s", c.Config.Domain, endpoint, query.Encode())
}

func (c *Client) get(endpoint string, query url.Values) (*response, error) {
	url := c.prepareURL(endpoint, query)
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	log.Println(res.Request.Method, url)

	var response response
	err = json.NewDecoder(res.Body).Decode(&response)
	log.Printf("Code: %d, Message: %q, RequestID: %s", response.Code, response.Message, response.RequestID)

	return &response, err
}

func (c *Client) post(endpoint string, payload interface{}, query url.Values) (*response, error) {
	message, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(message)
	url := c.prepareURL(endpoint, query)
	res, err := http.Post(url, "application/json", buf)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	log.Println(res.Request.Method, url)

	var response response
	err = json.NewDecoder(res.Body).Decode(&response)
	log.Printf("Code: %d, Message: %q, RequestID: %s", response.Code, response.Message, response.RequestID)

	return &response, err
}
