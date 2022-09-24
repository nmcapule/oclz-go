package opencart

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/tidwall/gjson"

	log "github.com/sirupsen/logrus"
)

type requestConfig struct {
}

type requestOption func(cfg *requestConfig)

func (c *Client) url(endpoint string, query url.Values) *url.URL {
	endpoint = strings.TrimPrefix(endpoint, "/")
	baseURL := fmt.Sprintf("%s%s", c.Config.Domain, endpoint)
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		baseURL = endpoint
	}
	u, err := url.Parse(fmt.Sprintf("%s&%s", baseURL, query.Encode()))
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

	// Override request with one derived from the original.
	login := &http.Request{
		Method: http.MethodPost,
		URL:    c.url("common/login", nil),
		Body: io.NopCloser(strings.NewReader(
			url.Values{
				"username": []string{c.Config.Username},
				"password": []string{c.Config.Password},
				"redirect": []string{req.URL.String()},
			}.Encode(),
		)),
		Header: map[string][]string{
			// "Content-Type": {"multipart/form-data"},
			"Content-Type": {"application/x-www-form-urlencoded"},
			"Accept":       {"*/*"},
		},
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	res, err := client.Do(login)
	if err != nil {
		return nil, fmt.Errorf("http request: %v", err)
	}

	// Manually do redirects, since Go doesn't do it for us. It should, but
	// it's not doing it right now :/
	req = &http.Request{
		URL:    c.url(res.Header.Get("location"), nil),
		Header: make(map[string][]string),
	}
	cookies := res.Cookies()
	for i := range cookies {
		req.AddCookie(cookies[i])
	}
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %v", err)
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %v", err)
	}
	gj := gjson.ParseBytes(b)
	log.Fatalln("body:", gj)
	return &gj, nil
}
