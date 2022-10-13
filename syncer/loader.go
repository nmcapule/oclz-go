package syncer

import (
	"encoding/json"
	"fmt"

	"github.com/nmcapule/oclz-go/integrations/intent"
	"github.com/nmcapule/oclz-go/integrations/lazada"
	"github.com/nmcapule/oclz-go/integrations/models"
	"github.com/nmcapule/oclz-go/integrations/opencart"
	"github.com/nmcapule/oclz-go/integrations/shopee"
	"github.com/nmcapule/oclz-go/integrations/tiktok"
	"github.com/nmcapule/oclz-go/oauth2"
	"github.com/pocketbase/pocketbase/daos"

	log "github.com/sirupsen/logrus"
)

// LoadClient loads a client depending on the config vendor.
func LoadClient(dao *daos.Dao, tenantName string) (models.IntegrationClient, error) {
	collection, err := dao.FindCollectionByNameOrId("tenants")
	if err != nil {
		return nil, err
	}
	record, err := dao.FindFirstRecordByData(collection, "name", tenantName)
	if err != nil {
		return nil, err
	}

	oauth2Service := &oauth2.Service{Dao: dao}
	tenant := models.TenantFrom(record)
	switch tenant.Vendor {
	case intent.Vendor:
		var config intent.Config
		err := json.Unmarshal(tenant.Config, &config)
		if err != nil {
			return nil, err
		}
		return &intent.Client{
			BaseDatabaseTenant: &models.BaseDatabaseTenant{
				BaseTenant: tenant,
				Dao:        dao,
			},
		}, nil
	case opencart.Vendor:
		var config opencart.Config
		err := json.Unmarshal(tenant.Config, &config)
		if err != nil {
			return nil, err
		}
		return &opencart.Client{
			BaseTenant: tenant,
			DatabaseTenant: &models.BaseDatabaseTenant{
				BaseTenant: tenant,
				Dao:        dao,
			},
			Config: &config,
		}, nil
	case tiktok.Vendor:
		var config tiktok.Config
		err := json.Unmarshal(tenant.Config, &config)
		if err != nil {
			return nil, err
		}
		credentials, err := oauth2Service.Load(tenant.ID)
		if err != nil {
			return nil, err
		}
		return &tiktok.Client{
			BaseTenant:  tenant,
			Config:      &config,
			Credentials: credentials,
		}, nil
	case lazada.Vendor:
		var config lazada.Config
		err := json.Unmarshal(tenant.Config, &config)
		if err != nil {
			return nil, err
		}
		credentials, err := oauth2Service.Load(tenant.ID)
		if err != nil {
			return nil, err
		}
		return &lazada.Client{
			BaseTenant:  tenant,
			Config:      &config,
			Credentials: credentials,
		}, nil
	case shopee.Vendor:
		var config shopee.Config
		err := json.Unmarshal(tenant.Config, &config)
		if err != nil {
			return nil, err
		}
		credentials, err := oauth2Service.Load(tenant.ID)
		if err == oauth2.ErrNoCredentials {
			log.Warnf("no credentials found for %s, anyways...", tenant.Name)
		} else if err != nil {
			return nil, err
		}
		return &shopee.Client{
			BaseTenant: tenant,
			DatabaseTenant: &models.BaseDatabaseTenant{
				BaseTenant: tenant,
				Dao:        dao,
			},
			Config:      &config,
			Credentials: credentials,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported vendor %q", tenant.Vendor)
	}
}
