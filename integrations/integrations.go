package integrations

import (
	"encoding/json"
	"fmt"

	"github.com/nmcapule/oclz-go/integrations/models"
	"github.com/nmcapule/oclz-go/integrations/oauth2"
	"github.com/nmcapule/oclz-go/integrations/tiktok"
	"github.com/pocketbase/pocketbase/daos"
)

// LoadClient loads a client depending on the config vendor.
func LoadClient(dao *daos.Dao, tenantName string) (models.VendorClient, error) {
	collection, err := dao.FindCollectionByNameOrId("tenants")
	if err != nil {
		return nil, err
	}
	record, err := dao.FindFirstRecordByData(collection, "name", tenantName)
	if err != nil {
		return nil, err
	}

	oauth2Service := &oauth2.Service{Dao: dao}

	tenant := record.GetId()
	vendor := record.GetStringDataValue("vendor")
	configRaw := record.GetStringDataValue("config")

	switch vendor {
	case tiktok.Vendor:
		var config tiktok.Config
		err := json.Unmarshal([]byte(configRaw), &config)
		if err != nil {
			return nil, err
		}
		credentials, err := oauth2Service.Load(tenant)
		if err != nil {
			return nil, err
		}

		return &tiktok.Client{
			Config:      &config,
			Credentials: credentials,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported vendor %q", vendor)
	}
}

// Syncer orchestrates how to sync items across multiple tenants.
type Syncer struct {
}
