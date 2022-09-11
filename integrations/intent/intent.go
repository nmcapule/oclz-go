package intent

import (
	"github.com/nmcapule/oclz-go/integrations/models"
	"github.com/nmcapule/oclz-go/oauth2"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
)

const Vendor = "DEFAULT"

type Config struct {
}

type Client struct {
	*models.BaseTenant
	Config *Config
	Dao    *daos.Dao
}

func (c *Client) CollectAllItems() ([]*models.Item, error) {
	collection, err := c.Dao.FindCollectionByNameOrId("tenant_inventory")
	if err != nil {
		return nil, err
	}
	inventory, err := c.Dao.FindRecordsByExpr(collection, dbx.HashExp{
		"tenant": c.ID,
	})
	if err != nil {
		return nil, err
	}
	var items []*models.Item
	for _, record := range inventory {
		items = append(items, models.ItemFrom(record))
	}
	return items, nil
}

func (c *Client) LoadItem(sellerSKU string) (*models.Item, error) {
	collection, err := c.Dao.FindCollectionByNameOrId("tenant_inventory")
	if err != nil {
		return nil, err
	}
	inventory, err := c.Dao.FindRecordsByExpr(collection, dbx.HashExp{
		"tenant":     c.ID,
		"seller_sku": sellerSKU,
	})
	if err != nil {
		return nil, err
	}
	if len(inventory) == 0 {
		return nil, models.ErrNotFound
	}
	if len(inventory) > 1 {
		return nil, models.ErrMultipleItems
	}
	return models.ItemFrom(inventory[0]), nil

}

func (c *Client) SaveItem(item *models.Item) error {
	collection, err := c.Dao.FindCollectionByNameOrId("tenant_inventory")
	if err != nil {
		return err
	}
	return c.Dao.SaveRecord(item.ToRecord(collection))
}

func (c *Client) CredentialsManager() oauth2.CredentialsManager {
	return nil
}
