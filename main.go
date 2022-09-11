package main

import (
	"time"

	"github.com/nmcapule/oclz-go/syncer"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	log "github.com/sirupsen/logrus"
)

func daemon(app *pocketbase.PocketBase) {
	syncer, err := syncer.NewSyncer(app.Dao(), "circuit.rocks")
	if err != nil {
		log.Fatalf("instantiate syncer: %v", err)
	}

	refreshInterval := 24 * time.Hour
	refreshAfter := time.Now()
	for {
		if time.Now().After(refreshAfter) {
			// log.Info("collect inventory...")
			// if err := syncer.CollectAllItems(); err != nil {
			// 	log.Fatalf("collect all live tenant items: %v", err)
			// }

			// item, err := syncer.Tenants["CIRCUIT_ROCKS_LAZADA"].LoadItem("2162")
			// if err != nil {
			// 	panic(err)
			// }
			// log.Infoln("%+v", item)

			log.Fatalln(syncer.Tenants["CIRCUIT_ROCKS_SHOPEE"].CredentialsManager().GenerateAuthorizationURL())

			// code := "7265774441464b68764164494f666978"
			// shopID := "20469516"
			// log.Fatalln(syncer.Tenants["CIRCUIT_ROCKS_SHOPEE"].(*shopee.Client).GenerateCredentials(code, shopID))

			// oauth2Service := oauth2.Service{Dao: syncer.Dao}
			// credentials, err := syncer.Tenants["CIRCUIT_ROCKS_SHOPEE"].(*shopee.Client).RefreshCredentials()
			// if err != nil {
			// 	panic(err)
			// }
			// err = oauth2Service.Save(credentials)
			// if err != nil {
			// 	panic(err)
			// }

			refreshAfter = refreshAfter.Add(refreshInterval)
			continue
		}

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
