package tiktok

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

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

func (c *Client) http(method string, endpoint string, query url.Values, payload interface{}) (*response, error) {
	message, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	url := c.prepareURL(endpoint, query)

	httpClient := http.DefaultClient
	request, err := http.NewRequest(method, url, bytes.NewBuffer(message))
	if err != nil {
		return nil, err
	}
	request.Header.Add("Content-Type", "application/json")

	res, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	log.WithFields(log.Fields{
		"method":  method,
		"url":     url,
		"payload": string(message),
	}).Infoln("HTTP Request", string(message))

	var response response
	err = json.NewDecoder(res.Body).Decode(&response)
	log.WithFields(log.Fields{
		"code":       response.Code,
		"message":    response.Message,
		"request_id": response.RequestID,
	}).Infoln("Received response")

	if response.Code != 0 {
		return &response, fmt.Errorf(response.Message)
	}
	return &response, err
}

func (c *Client) get(endpoint string, query url.Values) (*response, error) {
	return c.http(http.MethodGet, endpoint, query, nil)
}

func (c *Client) post(endpoint string, payload interface{}, query url.Values) (*response, error) {
	return c.http(http.MethodPost, endpoint, query, payload)
}

func (c *Client) put(endpoint string, payload interface{}, query url.Values) (*response, error) {
	return c.http(http.MethodPut, endpoint, query, payload)
}
