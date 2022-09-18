package syncer

import (
	"time"

	pbm "github.com/pocketbase/pocketbase/models"
)

type inventoryDelta struct {
	Inventory string    `json:"tenant_inventory"`
	Field     string    `json:"field"`
	NValue    float64   `json:"nvalue"`
	SValue    string    `json:"svalue"`
	Created   time.Time `json:"created"`
}

func (s *Syncer) saveInventoryDelta(delta *inventoryDelta) error {
	collection, err := s.Dao.FindCollectionByNameOrId("tenant_inventory_delta")
	if err != nil {
		return err
	}
	record := pbm.NewRecord(collection)
	record.SetDataValue("tenant_inventory", delta.Inventory)
	record.SetDataValue("field", delta.Field)
	record.SetDataValue("nvalue", delta.NValue)
	record.SetDataValue("svalue", delta.SValue)
	return s.Dao.SaveRecord(record)
}
