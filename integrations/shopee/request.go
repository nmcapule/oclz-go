package shopee

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	log "github.com/sirupsen/logrus"
)

const (
	codeOk        = ""
	codeCallLimit = "TODO(ncapule): Find out the code for this"
)

type requestConfig struct {
	stripAccessToken bool
	stripShopID      bool
}

type requestOption func(cfg *requestConfig)

func tokenRetrievalMode(cfg *requestConfig) {
	cfg.stripAccessToken = true
	cfg.stripShopID = true
}

func (c *Client) url(endpoint string, query url.Values) *url.URL {
	u, err := url.Parse(fmt.Sprintf("%s%s?%s", c.Config.Domain, endpoint, query.Encode()))
	if err != nil {
		log.WithFields(log.Fields{
			"domain":   c.Config.Domain,
			"endpoint": endpoint,
		}).Fatalln("Cannot parse URL")
	}
	return u
}

func (c *Client) request(req *http.Request, opts ...requestOption) (*gjson.Result, error) {
	var config requestConfig
	for _, opt := range opts {
		opt(&config)
	}
	timestamp := time.Now().Unix()

	// Harvest endpoint and query from request.
	baseURL, _ := url.Parse(c.Config.Domain)
	endpoint := strings.TrimPrefix(req.URL.Path, baseURL.Path)
	query := req.URL.Query()
	query.Set("partner_id", strconv.FormatInt(c.Config.PartnerID, 10))
	query.Set("timestamp", strconv.FormatInt(timestamp, 10))
	query.Set("sign", signature(c.Config, endpoint, timestamp))
	if !config.stripAccessToken {
		query.Set("access_token", c.Credentials.AccessToken)
	}
	if !config.stripShopID {
		query.Set("shop_id", strconv.FormatInt(c.Config.ShopID, 10))
	}
	req.URL.RawQuery = query.Encode()

	retry := 3
	backoff := 1
	var gres gjson.Result
	for retry > 0 {
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("http request: %v", err)
		}
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("read body: %v", err)
		}
		gres = gjson.ParseBytes(b)
		if gres.Get("warning").String() != codeOk {
			log.Warningf("Shopee request warning: %s", gres.Get("warning").String())
		}
		if gres.Get("error").String() == codeOk {
			break
		}
		if gres.Get("error").String() == codeCallLimit {
			log.Warnln("api call exceeded, retrying...")
			time.Sleep(time.Duration(backoff) * time.Second)
			retry -= 1
			backoff *= 2
			continue
		}
		return nil, fmt.Errorf(
			"%s: %s, %s",
			gres.Get("request_id").String(),
			gres.Get("error").String(), gres.Get("message"))
	}
	return &gres, nil
}

func signature(config *Config, endpoint string, timestamp int64) string {
	base := fmt.Sprintf("%d%s%d", config.PartnerID, endpoint, timestamp)
	h := hmac.New(sha256.New, []byte(config.PartnerKey))
	if _, err := h.Write([]byte(base)); err != nil {
		log.Fatalf("Failed to hash %q: %v", base, err)
	}
	return hex.EncodeToString(h.Sum(nil))
}
