package main

import (
	"log"

	"github.com/nmcapule/oclz-go/integrations"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	app := pocketbase.New()

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		go func(app *pocketbase.PocketBase) {
			syncer, err := integrations.LoadClient(app.Dao(), "CIRCUIT_ROCKS_TIKTOK")
			if err != nil {
				log.Fatal(err)
			}

			syncer.CollectAllItems()
		}(app)

		return nil
	})

	log.Fatal(app.Start())
}
