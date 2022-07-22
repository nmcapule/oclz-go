package main

import (
	"github.com/nmcapule/oclz-go/integrations"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	log "github.com/sirupsen/logrus"
)

func main() {
	app := pocketbase.New()

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		go func(app *pocketbase.PocketBase) {
			syncer, err := integrations.NewSyncer(app.Dao(), "circuit.rocks")
			if err != nil {
				log.Fatal(err)
			}

			if err := syncer.CollectAllItems(); err != nil {
				log.Fatal(err)
			}
		}(app)

		return nil
	})

	log.Fatal(app.Start())
}
