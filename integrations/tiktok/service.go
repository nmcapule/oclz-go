package tiktok

import "github.com/nmcapule/oclz-go/integrations/models"

func (c *Client) BackgroundService() models.BackgroundService {
	return c
}

func (c *Client) Start() error {
	return nil
}
