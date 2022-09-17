package lazada

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
	endpoint := "https://auth.lazada.com/oauth/authorize"

	return c.url(endpoint, url.Values{
		"response_type": []string{"code"},
		"force_auth":    []string{"true"},
		"redirect_uri":  []string{c.Config.RedirectURI},
		"client_id":     []string{c.Config.AppKey},
	}).String()
}

func (c *Client) GenerateCredentials(greq gjson.Result) (*oauth2.Credentials, error) {
	gres, err := c.request(&http.Request{
		Method: http.MethodGet,
		URL: c.url("https://auth.lazada.com/rest/auth/token/create", url.Values{
			"code": []string{greq.Get("code").String()},
		}),
	}, tokenRetrievalMode)
	if err != nil {
		return nil, fmt.Errorf("error response: %v", err)
	}

	c.Credentials = &oauth2.Credentials{
		Tenant:       c.ID,
		AccessToken:  gres.Get("access_token").String(),
		RefreshToken: gres.Get("refresh_token").String(),
		Expires:      time.Now().Add(time.Duration(gres.Get("expires_in").Int()) * time.Second),
	}

	return c.Credentials, err
}

func (c *Client) RefreshCredentials() (*oauth2.Credentials, error) {
	gres, err := c.request(&http.Request{
		Method: http.MethodGet,
		URL: c.url("https://auth.lazada.com/rest/auth/token/refresh", url.Values{
			"refresh_token": []string{c.Credentials.RefreshToken},
		}),
	}, tokenRetrievalMode)
	if err != nil {
		return nil, fmt.Errorf("error response: %v", err)
	}

	c.Credentials = &oauth2.Credentials{
		Tenant:       c.ID,
		AccessToken:  gres.Get("access_token").String(),
		RefreshToken: gres.Get("refresh_token").String(),
		Expires:      time.Now().Add(time.Duration(gres.Get("expires_in").Int()) * time.Second),
	}

	return c.Credentials, err
}

func (c *Client) CredentialsExpiry() time.Time {
	return c.Credentials.Expires
}
