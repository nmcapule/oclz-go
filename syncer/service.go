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
		job := s.Tenants[i].BackgroundService()
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

	// go scheduler.Launch(func(quit chan struct{}) {
	// 	log.Infoln("collect inventory...")
	// 	if err := s.CollectAllItems(); err != nil {
	// 		log.Fatalf("collect all live tenant items: %v", err)
	// 	}
	// }, scheduler.LoopConfig{InitialWait: 5 * time.Second, RetryWait: 24 * time.Hour})

	go scheduler.Loop(func(quit chan struct{}) {
		log.Infoln("refreshing oauth2 credentials...")
		if err := s.RefreshCredentials(); err != nil {
			log.Fatalf("refreshing all tenants credentials: %v", err)
		}
	}, scheduler.LoopConfig{RetryWait: 30 * time.Minute})

	return scheduler.Loop(func(quit chan struct{}) {
		log.Info("sync inventory...")
		items, err := s.IntentTenant.CollectAllItems()
		if err != nil {
			log.Fatalf("collect all intent items: %v", err)
		}
		for _, item := range items {
			err := s.SyncItem(item.SellerSKU)
			if err != nil {
				log.Fatalf("syncing %q: %v", item.SellerSKU, err)
			}
		}
	}, scheduler.LoopConfig{RetryWait: 1 * time.Hour})
}
