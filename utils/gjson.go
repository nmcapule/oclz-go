package utils

import (
	"encoding/json"
	"log"

	"github.com/tidwall/gjson"
)

// GJSONFrom converts Go structs into gjson.Result. Panics on failure.
func GJSONFrom(v any) *gjson.Result {
	b, err := json.Marshal(v)
	if err != nil {
		log.Fatalf("serializing JSON: %v", err)
	}
	data := gjson.ParseBytes(b)
	return &data
}
