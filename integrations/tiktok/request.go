package tiktok

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

const (
	messageOk = "Success"
)

type requestConfig struct {
	tokenRetrievalMode bool
}

type requestOption func(cfg *requestConfig)

func tokenRetrievalMode(cfg *requestConfig) {
	cfg.tokenRetrievalMode = true
}

func (c *Client) url(endpoint string, query url.Values) *url.URL {
	baseURL := fmt.Sprintf("%s%s", c.Config.Domain, endpoint)
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		baseURL = endpoint
	}
	u, err := url.Parse(fmt.Sprintf("%s?%s", baseURL, query.Encode()))
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
	query.Set("app_key", c.Config.AppKey)

	if !config.tokenRetrievalMode {
		query.Set("timestamp", fmt.Sprintf("%d", timestamp))
		query.Set("shop_id", c.Config.ShopID)
		// Sign before setting access token.
		query.Set("sign", signature(c.Config, endpoint, query))
		query.Set("access_token", c.Credentials.AccessToken)
	}
	req.URL.RawQuery = query.Encode()

	retry := 3
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
		if strings.EqualFold(gres.Get("message").String(), messageOk) {
			break
		}
		return nil, fmt.Errorf(
			"%s: %s, %s",
			gres.Get("request_id").String(),
			gres.Get("code").String(), gres.Get("message"))
	}
	return &gres, nil
}

func signature(config *Config, endpoint string, query url.Values) string {
	var queryConcats []string
	for k := range query {
		queryConcats = append(queryConcats, fmt.Sprintf("%s%s", k, query.Get(k)))
	}
	sort.Strings(queryConcats)

	base := fmt.Sprintf(
		"%s%s%s%s",
		config.AppSecret,
		endpoint,
		strings.Join(queryConcats, ""),
		config.AppSecret,
	)
	h := hmac.New(sha256.New, []byte(config.AppSecret))
	if _, err := h.Write([]byte(base)); err != nil {
		log.Fatalf("encoding signature of %q: %v", base, err)
	}
	return hex.EncodeToString(h.Sum(nil))
}
