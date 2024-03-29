package lazada

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	log "github.com/sirupsen/logrus"
)

const (
	codeOk        = "0"
	codeCallLimit = "ApiCallLimit"
)

type requestConfig struct {
	stripAccessToken bool
}

type requestOption func(cfg *requestConfig)

func tokenRetrievalMode(cfg *requestConfig) {
	cfg.stripAccessToken = true
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

	// Harvest endpoint and query from request.
	baseURL, _ := url.Parse(c.Config.Domain)
	endpoint := strings.TrimPrefix(req.URL.Path, baseURL.Path)

	query := req.URL.Query()
	query.Set("app_key", c.Config.AppKey)
	query.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	if !config.stripAccessToken {
		query.Set("access_token", c.Credentials.AccessToken)
	}
	query.Set("sign_method", "sha256")
	if req.Method == http.MethodGet {
		query.Set("sign", signature(c.Config.AppSecret, endpoint, query))
		req.URL.RawQuery = query.Encode()
	} else if req.Method == http.MethodPost {
		// If method is POST, attach the usual query params to the body.
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("read body: %v", err)
		}
		parsedQuery, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, fmt.Errorf("parse body: %v", err)
		}
		for key, values := range parsedQuery {
			for _, v := range values {
				query.Add(key, v)
			}
		}
		query.Set("sign", signature(c.Config.AppSecret, endpoint, query))
		req.Body = io.NopCloser(strings.NewReader(query.Encode()))
	}

	retry := 3
	backoff := 1
	var gj gjson.Result
	for retry > 0 {
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("http request: %v", err)
		}
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("read body: %v", err)
		}
		gj = gjson.ParseBytes(b)
		if gj.Get("code").String() == codeOk {
			break
		}
		if gj.Get("code").String() == codeCallLimit {
			log.Warnln("api call exceeded, retrying...")
			time.Sleep(time.Duration(backoff) * time.Second)
			retry -= 1
			backoff *= 2
			continue
		}
		return nil, fmt.Errorf("%s, %s", gj.Get("code").String(), gj.Get("message"))
	}
	return &gj, nil
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
