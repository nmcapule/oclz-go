package tiktok

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/nmcapule/oclz-go/oauth2"
	"github.com/tidwall/gjson"
)

func (c *Client) CredentialsManager() oauth2.CredentialsManager {
	return c
}

func (c *Client) GenerateAuthorizationURL() string {
	endpoint := "https://auth.tiktok-shops.com/oauth/authorize"

	// TODO: state must be randomized as per documentation:
	// https://developers.tiktok-shops.com/documents/document/234120
	return c.url(endpoint, url.Values{
		"app_key": []string{c.Config.AppKey},
		"state":   []string{c.Config.AppKey},
	}).String()
}

func (c *Client) GenerateCredentials(greq gjson.Result) (*oauth2.Credentials, error) {
	gres, err := c.request(&http.Request{
		Method: http.MethodGet,
		URL: c.url("https://auth.tiktok-shops.com/api/v2/token/get", url.Values{
			"app_secret": []string{c.Config.AppSecret},
			"auth_code":  []string{greq.Get("code").String()},
			"grant_type": []string{"authorized_code"},
		}),
	}, tokenRetrievalMode)
	if err != nil {
		return nil, fmt.Errorf("error response: %v", err)
	}

	c.Credentials = &oauth2.Credentials{
		Tenant:       c.ID,
		AccessToken:  gres.Get("data.access_token").String(),
		RefreshToken: gres.Get("data.refresh_token").String(),
		Expires:      time.Unix(gres.Get("data.access_token_expire_in").Int(), 0),
	}

	return c.Credentials, err
}

func (c *Client) RefreshCredentials() (*oauth2.Credentials, error) {
	gres, err := c.request(&http.Request{
		Method: http.MethodGet,
		URL: c.url("https://auth.tiktok-shops.com/api/v2/token/refresh", url.Values{
			"app_secret":    []string{c.Config.AppSecret},
			"refresh_token": []string{c.Credentials.RefreshToken},
			"grant_type":    []string{"refresh_token"},
		}),
	}, tokenRetrievalMode)
	if err != nil {
		return nil, fmt.Errorf("error response: %v", err)
	}

	c.Credentials = &oauth2.Credentials{
		Tenant:       c.ID,
		AccessToken:  gres.Get("data.access_token").String(),
		RefreshToken: gres.Get("data.refresh_token").String(),
		Expires:      time.Unix(gres.Get("data.access_token_expire_in").Int(), 0),
	}

	return c.Credentials, err
}

func (c *Client) CredentialsExpiry() time.Time {
	return c.Credentials.Expires
}
