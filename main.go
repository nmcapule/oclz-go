package main

import (
	"github.com/nmcapule/oclz-go/syncer"
	"github.com/nmcapule/oclz-go/views"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	log "github.com/sirupsen/logrus"
)

func main() {
	app := pocketbase.New()
	noSync := app.RootCmd.PersistentFlags().Bool("nosync", true, "Set to true to deactivate syncing.")

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		syncer, err := syncer.NewSyncer(app.Dao(), "circuit.rocks")
		if err != nil {
			log.Fatalf("instantiate syncer: %v", err)
		}

		// Set up custom routes using root view.
		routes := views.RootView{
			App:    app,
			Syncer: syncer,
		}
		if err := routes.Hook(e.Router); err != nil {
			log.Fatalf("Hooking custom routes: %w", err)
		}

		// If we're not supposed to sync, just return.
		if *noSync {
			return nil
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
