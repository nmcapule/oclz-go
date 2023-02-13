package oauth2

import (
	"errors"
	"fmt"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

var (
	ErrNoCredentials       = errors.New("no oauth2 credentials found")
	ErrMultipleCredentials = errors.New("unexpected multiple oauth2 credentials found")
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
	record, err := s.Dao.FindFirstRecordByData("tenant_oauth2", "tenant", tenant)
	if err != nil {
		return nil, ErrNoCredentials
	}
	return &Credentials{
		Tenant:       tenant,
		AccessToken:  record.GetString("access_token"),
		RefreshToken: record.GetString("refresh_token"),
		// Expires:      record.GetTime("expires"),
		// Created:      record.GetTime("created"),
		// Updated:      record.GetTime("updated"),
		// TODO(nmcapule): Above doesn't work, so we do a workaround.
		Expires: record.GetDateTime("expires").Time(),
		Created: record.GetDateTime("created").Time(),
		Updated: record.GetDateTime("updated").Time(),
	}, nil
}

func (s *Service) Save(credentials *Credentials) error {
	collection, err := s.Dao.FindCollectionByNameOrId("tenant_oauth2")
	if err != nil {
		return err
	}
	records, err := s.Dao.FindRecordsByExpr(collection.Name, dbx.HashExp{
		"tenant": credentials.Tenant,
	})
	if err != nil {
		return fmt.Errorf("retrieving existing creds: %v", err)
	}
	if len(records) > 1 {
		return ErrMultipleCredentials
	}

	var record *models.Record
	if len(records) == 1 {
		record = records[0]
	} else {
		record = models.NewRecord(collection)
	}
	record.Set("tenant", credentials.Tenant)
	record.Set("access_token", credentials.AccessToken)
	record.Set("refresh_token", credentials.RefreshToken)
	record.Set("expires", credentials.Expires)
	return s.Dao.Save(record)
}
