package main

import (
	"time"

	"github.com/nmcapule/oclz-go/syncer"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	log "github.com/sirupsen/logrus"
)

type loopConfig struct {
	initialWait time.Duration
	retryWait   time.Duration
}

func launchLoop(fn func(quit chan struct{}), config loopConfig) error {
	time.Sleep(config.initialWait)
	quit := make(chan struct{})
	fn(quit)
	ticker := time.NewTicker(config.retryWait)
	for {
		select {
		case <-ticker.C:
			fn(quit)
		case <-quit:
			return nil
		}
	}
}

func daemon(app *pocketbase.PocketBase) {
	syncer, err := syncer.NewSyncer(app.Dao(), "circuit.rocks")
	if err != nil {
		log.Fatalf("instantiate syncer: %v", err)
	}

	// go launchLoop(func(quit chan struct{}) {
	// 	log.Infoln("collect inventory...")
	// 	if err := syncer.CollectAllItems(); err != nil {
	// 		log.Fatalf("collect all live tenant items: %v", err)
	// 	}
	// }, loopConfig{initialWait: 5 * time.Second, retryWait: 24 * time.Hour})
	go launchLoop(func(quit chan struct{}) {
		log.Infoln("refreshing oauth2 credentials...")
		if err := syncer.RefreshCredentials(); err != nil {
			log.Fatalf("refreshing all tenants credentials: %v", err)
		}
	}, loopConfig{retryWait: 1 * time.Hour})

	// launchLoop(func(quit chan struct{}) {
	// 	log.Info("sync inventory...")
	// 	items, err := syncer.IntentTenant.CollectAllItems()
	// 	if err != nil {
	// 		log.Fatalf("collect all intent items: %v", err)
	// 	}
	// 	for _, item := range items {
	// 		err := syncer.SyncItem(item.SellerSKU)
	// 		if err != nil {
	// 			log.Fatalf("syncing %q: %v", item.SellerSKU, err)
	// 		}
	// 	}
	// }, loopConfig{retryWait: 5 * time.Second})
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
