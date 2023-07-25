package main

import (
	"os"

	"github.com/nmcapule/oclz-go/syncer"
	"github.com/nmcapule/oclz-go/views"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	log "github.com/sirupsen/logrus"
)

func main() {
	app := pocketbase.New()

	flags := app.RootCmd.PersistentFlags()

	noSync := flags.Bool("nosync", true, "Set to true to deactivate syncing.")
	publicDir := flags.String("public", "./static", "Directory to serve static files")

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// serves static files from the provided public dir (if exists)
		e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS(*publicDir), false))

		return nil
	})
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
