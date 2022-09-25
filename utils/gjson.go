package utils

import (
	"encoding/json"
	"log"

	"github.com/tidwall/gjson"
)

func GJSONFrom(v any) gjson.Result {
	b, err := json.Marshal(v)
	if err != nil {
		log.Fatalf("serializing JSON: %v", err)
	}
	return gjson.ParseBytes(b)
}
