package main

import (
	"github.com/nmcapule/oclz-go/syncer"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	log "github.com/sirupsen/logrus"
)

func main() {
	app := pocketbase.New()
	noSync := app.RootCmd.PersistentFlags().Bool("nosync", true, "Set to true to deactivate syncing.")

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		if *noSync {
			return nil
		}
		syncer, err := syncer.NewSyncer(app.Dao(), "circuit.rocks", syncer.Config{
			ContinueOnSyncItemError: true,
		})
		if err != nil {
			log.Fatalf("instantiate syncer: %v", err)
		}
		go func() {
			log.Infoln("Syncer background service has started.")
			err := syncer.Start()
			if err != nil {
				log.Fatalf("Syncer background service unexpectedly exited: %v", err)
			}
			log.Infoln("Syncer background service has finished.")
		}()
		return nil
	})

	log.Fatal(app.Start())
}
