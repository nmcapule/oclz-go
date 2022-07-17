package oauth2

import (
	"time"

	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

type Credentials struct {
	Tenant       string
	AccessToken  string
	RefreshToken string
	Expires      time.Time
	Created      time.Time
	Updated      time.Time
}

type Service struct {
	Dao *daos.Dao
}

func (s *Service) Load(tenant string) (*Credentials, error) {
	collection, err := s.Dao.FindCollectionByNameOrId("tenant_oauth2")
	if err != nil {
		return nil, err
	}
	record, err := s.Dao.FindFirstRecordByData(collection, "tenant", tenant)
	if err != nil {
		return nil, err
	}
	return &Credentials{
		Tenant:       tenant,
		AccessToken:  record.GetStringDataValue("access_token"),
		RefreshToken: record.GetStringDataValue("refresh_token"),
		Expires:      record.GetTimeDataValue("expires"),
		Created:      record.GetTimeDataValue("created"),
		Updated:      record.GetTimeDataValue("updated"),
	}, nil
}

func (s *Service) Save(credentials *Credentials) error {
	collection, err := s.Dao.FindCollectionByNameOrId("tenant_oauth2")
	if err != nil {
		return err
	}
	record := models.NewRecord(collection)
	record.SetDataValue("tenant", credentials.Tenant)
	record.SetDataValue("access_token", credentials.AccessToken)
	record.SetDataValue("refresh_token", credentials.RefreshToken)
	record.SetDataValue("expires", credentials.Expires)
	return s.Dao.Save(record)
}
