package tiktok

import "github.com/nmcapule/oclz-go/integrations/models"

func (c *Client) Daemon() models.Daemon {
	return c
}

func (c *Client) Start() error {
	return nil
}
