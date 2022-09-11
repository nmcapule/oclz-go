package shopee

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nmcapule/oclz-go/oauth2"
	"github.com/tidwall/gjson"
)

func (c *Client) CredentialsManager() oauth2.CredentialsManager {
	return c
}

func (c *Client) GenerateAuthorizationURL() string {
	const endpoint = "/api/v2/shop/auth_partner"

	timestamp := time.Now().Unix()

	return c.url(endpoint, url.Values{
		"partner_id": []string{strconv.FormatInt(c.Config.PartnerID, 10)},
		"timestamp":  []string{strconv.FormatInt(timestamp, 10)},
		"sign":       []string{signature(c.Config, endpoint, timestamp)},
		"redirect":   []string{"https://n8n.nmcapule.dev/webhook/circuit-rocks/shopee"},
	}).String()
}

func (c *Client) GenerateCredentials(greq gjson.Result) (*oauth2.Credentials, error) {
	body, err := json.Marshal(map[string]interface{}{
		"code":       greq.Get("code").String(),
		"shop_id":    c.Config.ShopID,
		"partner_id": c.Config.PartnerID,
	})
	if err != nil {
		return nil, fmt.Errorf("compose payload: %v", err)
	}

	gres, err := c.request(&http.Request{
		Method: http.MethodPost,
		URL:    c.url("/api/v2/auth/token/get", nil),
		Body:   io.NopCloser(strings.NewReader(string(body))),
	})
	if err != nil {
		return nil, fmt.Errorf("error response: %v", err)
	}

	c.Credentials = &oauth2.Credentials{
		Tenant:       c.ID,
		AccessToken:  gres.Get("access_token").String(),
		RefreshToken: gres.Get("refresh_token").String(),
		Expires:      time.Now().Add(time.Duration(gres.Get("expire_in").Int()) * time.Second),
	}

	return c.Credentials, err
}

func (c *Client) RefreshCredentials() (*oauth2.Credentials, error) {
	body, err := json.Marshal(map[string]interface{}{
		"shop_id":       c.Config.ShopID,
		"refresh_token": c.Credentials.RefreshToken,
		"partner_id":    c.Config.PartnerID,
	})
	if err != nil {
		return nil, fmt.Errorf("compose payload: %v", err)
	}

	gres, err := c.request(&http.Request{
		Method: http.MethodPost,
		URL:    c.url("/api/v2/auth/access_token/get", nil),
		Body:   io.NopCloser(strings.NewReader(string(body))),
	})
	if err != nil {
		return nil, fmt.Errorf("error response: %v", err)
	}

	c.Credentials.AccessToken = gres.Get("access_token").String()
	c.Credentials.RefreshToken = gres.Get("refresh_token").String()
	c.Credentials.Expires = time.Now().Add(time.Duration(gres.Get("expire_in").Int()) * time.Second)

	return c.Credentials, nil
}
