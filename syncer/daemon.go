package syncer

import (
	"time"

	"github.com/nmcapule/oclz-go/integrations/models"
	"github.com/nmcapule/oclz-go/utils/scheduler"

	log "github.com/sirupsen/logrus"
)

// Start starts the syncer's background service.
func (s *Syncer) Start() error {
	for i := range s.Tenants {
		job := s.Tenants[i].Daemon()
		if job == nil {
			continue
		}

		go func(tenant models.IntegrationClient) {
			log.WithFields(log.Fields{
				"tenant": tenant.Tenant().Name,
			}).Infoln("Background job has started")
			if err := job.Start(); err != nil {
				log.WithFields(log.Fields{
					"tenant": tenant.Tenant().Name,
				}).Fatalf("Background job has unexpectedly halted: %v", err)
			}
			log.WithFields(log.Fields{
				"tenant": tenant.Tenant().Name,
			}).Infoln("Background job has finished")
		}(s.Tenants[i])
	}

	go scheduler.Loop(func(quit chan struct{}) {
		log.Infoln("Start collecting inventory from all tenants...")
		if s.IntentTenant == nil {
			log.Warnf("Skipping item collection. No active intent tenant.")
			return
		}
		if err := s.CollectAllItems(); err != nil {
			log.Fatalf("Collect all live tenant items: %v", err)
		}
	}, scheduler.LoopConfig{RetryWait: 24 * time.Hour})

	go scheduler.Loop(func(quit chan struct{}) {
		log.Infoln("Refreshing oauth2 credentials of all tenants...")
		if err := s.RefreshCredentials(); err != nil {
			log.Fatalf("Refreshing all tenants credentials: %v", err)
		}
	}, scheduler.LoopConfig{RetryWait: 30 * time.Minute})

	return scheduler.Loop(func(quit chan struct{}) {
		log.Info("Sync inventory...")
		items, err := s.IntentTenant.CollectAllItems()
		if err != nil {
			log.Fatalf("Collect all intent items: %v", err)
		}
		for i, item := range items {
			log.WithFields(log.Fields{
				"seller_sku": item.SellerSKU,
				"index":      i,
				"total":      len(items),
			}).Debugln("Syncing item")
			err := s.SyncItem(item.SellerSKU)
			if err != nil {
				log.Fatalf("Syncing %q: %v", item.SellerSKU, err)
			}
		}
	}, scheduler.LoopConfig{InitialWait: 1 * time.Hour, RetryWait: 3 * time.Hour})
}
