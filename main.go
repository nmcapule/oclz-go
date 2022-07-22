package main

import (
	"github.com/nmcapule/oclz-go/syncer"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	log "github.com/sirupsen/logrus"
)

func daemon(app *pocketbase.PocketBase) {
	syncer, err := syncer.NewSyncer(app.Dao(), "circuit.rocks")
	if err != nil {
		log.Fatal("intantiate syncer: %v", err)
	}

	if err := syncer.CollectAllItems(); err != nil {
		log.Fatal("collect all live tenant items: %v", err)
	}

	intentItems, err := syncer.IntentTenant.CollectAllItems()
	if err != nil {
		log.Fatalf("collect all intent items: %v", err)
	}
	for _, item := range intentItems {
		err := syncer.SyncItem(item.SellerSKU)
		if err != nil {
			log.Fatalf("syncing %q: %v", item.SellerSKU, err)
		}
	}
}

func main() {
	app := pocketbase.New()

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		go daemon(app)
		return nil
	})

	log.Fatal(app.Start())
}
