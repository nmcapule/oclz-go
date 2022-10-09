package syncer

import (
	"fmt"

	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"

	log "github.com/sirupsen/logrus"
)

// LogHook is a logrus hook that writes the logs to the database.
type LogHook struct {
	LogLevels  []log.Level
	Dao        *daos.Dao
	collection *models.Collection
}

func (h *LogHook) Fire(entry *log.Entry) error {
	if h.collection == nil {
		collection, err := h.Dao.FindCollectionByNameOrId("custom_logs")
		if err != nil {
			return fmt.Errorf("retrieving collection: %s", err)
		}
		h.collection = collection
	}
	record := models.NewRecord(h.collection)
	record.SetDataValue("message", entry.Message)
	record.SetDataValue("fields", entry.Data)
	record.SetDataValue("level", entry.Level)
	record.SetDataValue("caller", entry.Caller.Function)
	return h.Dao.SaveRecord(record)
}

// Levels define on which log levels this hook would trigger
func (h *LogHook) Levels() []log.Level {
	return h.LogLevels
}
