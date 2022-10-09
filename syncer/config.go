package syncer

import (
	"github.com/nmcapule/oclz-go/utils"
	"github.com/pocketbase/pocketbase/models"
)

// Config contains configurable behavior flags for the syncer.
type Config struct {
	ContinueOnSyncItemError bool
}

func (s *Syncer) loadConfigFromGroup(group *models.Record) error {
	data := utils.GJSONFrom(group.GetDataValue("config"))
	s.Config.ContinueOnSyncItemError = data.Get("continue_on_sync_item_error").Bool()
	return nil
}
