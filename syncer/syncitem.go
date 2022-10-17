package syncer

import (
	"fmt"

	"github.com/nmcapule/oclz-go/integrations/models"

	log "github.com/sirupsen/logrus"
)

// SyncItem tries to sync a single seller sku across all tenants.
func (s *Syncer) SyncItem(sellerSKU string) error {
	tenantLiveItemMap := make(map[string]*models.Item)
	var totalDelta int
	for _, tenant := range s.Tenants {
		cached, err := s.tenantInventory(tenant.Tenant().Name, sellerSKU)
		if err == models.ErrNotFound {
			log.WithFields(log.Fields{
				"seller_sku": sellerSKU,
				"tenant":     tenant.Tenant().Name,
			}).Debugln("Item not found")
			continue
		}
		if err != nil {
			return fmt.Errorf("loading cached item %q from %s: %v", sellerSKU, tenant.Tenant().Name, err)
		}

		live, err := tenant.LoadItem(sellerSKU)
		if err != nil {
			if s.Config.ContinueOnSyncItemError {
				log.WithFields(log.Fields{
					"seller_sku": sellerSKU,
					"tenant":     tenant.Tenant().Name,
					"error":      err.Error(),
				}).Errorln("Failed to load item info. Skipping.")
				continue
			}
			return fmt.Errorf("loading live item %q from %s: %v", sellerSKU, tenant.Tenant().Name, err)
		}

		totalDelta += live.Stocks - cached.Stocks
		if live.Stocks != cached.Stocks {
			log.WithFields(log.Fields{
				"seller_sku": sellerSKU,
				"tenant":     tenant.Tenant().Name,
				"previous":   cached.Stocks,
				"stocks":     live.Stocks,
			}).Infoln("Pull update from live item stocks")
		}

		live.ID = cached.ID
		live.Created = cached.Created
		tenantLiveItemMap[tenant.Tenant().Name] = live

		// Pre-save the live item to the database.
		if err := s.saveTenantInventory(tenant.Tenant().Name, live); err != nil {
			return fmt.Errorf("saving cached item %q from %s: %v", sellerSKU, tenant.Tenant().Name, err)
		}
	}

	targetStocks := tenantLiveItemMap[s.IntentTenant.Tenant().Name].Stocks
	targetStocks += totalDelta
	if targetStocks < 0 {
		log.Warnf("warning: %s has negative stocks, setting to 0", sellerSKU)
		targetStocks = 0
	}

	for _, tenant := range s.Tenants {
		live, ok := tenantLiveItemMap[tenant.Tenant().Name]
		if !ok {
			log.WithFields(log.Fields{
				"seller_sku": sellerSKU,
				"tenant":     tenant.Tenant().Name,
			}).Debugln("Skip item sync, does not exist in tenant")
			continue
		}
		// Skip update if has the same stock as the intent tenant.
		if live.Stocks == targetStocks {
			continue
		}

		log.WithFields(log.Fields{
			"seller_sku": sellerSKU,
			"tenant":     tenant.Tenant().Name,
			"previous":   live.Stocks,
			"stocks":     targetStocks,
		}).Infoln("Push update to live item stocks")

		live.Stocks = targetStocks

		if err := tenant.SaveItem(live); err != nil {
			if s.Config.ContinueOnSyncItemError {
				log.WithFields(log.Fields{
					"seller_sku": sellerSKU,
					"tenant":     tenant.Tenant().Name,
					"error":      err.Error(),
				}).Errorln("Failed to save item info. Skipping.")
				continue
			}
			return fmt.Errorf("saving live item %q from %s: %v", sellerSKU, tenant.Tenant().Name, err)
		}
		if err := s.saveTenantInventory(tenant.Tenant().Name, live); err != nil {
			return fmt.Errorf("saving cached item %q from %s: %v", sellerSKU, tenant.Tenant().Name, err)
		}
	}

	return nil
}
