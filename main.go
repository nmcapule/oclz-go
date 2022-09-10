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

	// refreshInterval := 24 * time.Hour
	// refreshAfter := time.Now()
	for {
		item, err := syncer.Tenants["CIRCUIT_ROCKS_LAZADA"].LoadItem("2162")
		if err != nil {
			panic(err)
		}
		log.Infoln("%+v", item)
		// if time.Now().After(refreshAfter) {
		// 	log.Info("collect inventory...")
		// 	if err := syncer.CollectAllItems(); err != nil {
		// 		log.Fatal("collect all live tenant items: %v", err)
		// 	}
		// 	refreshAfter = refreshAfter.Add(refreshInterval)
		// 	continue
		// }

		// log.Info("sync inventory...")
		// items, err := syncer.IntentTenant.CollectAllItems()
		// if err != nil {
		// 	log.Fatalf("collect all intent items: %v", err)
		// }
		// for _, item := range items {
		// 	err := syncer.SyncItem(item.SellerSKU)
		// 	if err != nil {
		// 		log.Fatalf("syncing %q: %v", item.SellerSKU, err)
		// 	}
		// 	time.Sleep(500 * time.Millisecond)
		// }
	}
}

func main() {
	app := pocketbase.New()
	noSync := app.RootCmd.PersistentFlags().Bool("nosync", true, "Set to true to deactivate syncing.")

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		if *noSync {
			return nil
		}
		go daemon(app)
		return nil
	})

	log.Fatal(app.Start())
}
