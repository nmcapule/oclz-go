package syncer

import (
	"fmt"
	"strconv"

	"github.com/nmcapule/oclz-go/integrations/models"

	log "github.com/sirupsen/logrus"
)

// SyncItem tries to sync a single seller sku across all tenants.
func (s *Syncer) SyncItem(sellerSKU string) error {
	tenantItemMap := make(map[string]*models.Item)
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
		current, err := tenant.LoadItem(sellerSKU)
		if err != nil {
			return fmt.Errorf("loading live item %q from %s: %v", sellerSKU, tenant.Tenant().Name, err)
		}
		totalDelta += current.Stocks - cached.Stocks
		log.WithFields(log.Fields{
			"tenant":     tenant.Tenant().Name,
			"seller_sku": sellerSKU,
		}).Debugln("Load fresh item")

		if totalDelta != 0 {
			err := s.saveInventoryDelta(&inventoryDelta{
				Inventory: cached.ID,
				Field:     "stocks",
				NValue:    float64(current.Stocks),
				SValue:    strconv.Itoa(current.Stocks),
			})
			if err != nil {
				return fmt.Errorf("saving inventory delta for %s (%s): %v", sellerSKU, cached.ID, err)
			}
		}

		current.ID = cached.ID
		current.Created = cached.Created
		tenantItemMap[tenant.Tenant().Name] = current
	}

	targetStocks := tenantItemMap[s.IntentTenant.Tenant().Name].Stocks
	targetStocks += totalDelta
	if targetStocks < 0 {
		log.Warnf("warning: %s has negative stocks, setting to 0", sellerSKU)
		targetStocks = 0
	}

	for _, tenant := range s.Tenants {
		item, ok := tenantItemMap[tenant.Tenant().Name]
		if !ok {
			log.WithFields(log.Fields{
				"seller_sku": sellerSKU,
				"tenant":     tenant.Tenant().Name,
			}).Debugln("Skip item sync, does not exist in tenant")
			continue
		}
		// Skip update if has the same stock as the intent tenant.
		if item.Stocks == targetStocks {
			continue
		}
		item.Stocks = targetStocks
		if err := tenant.SaveItem(item); err != nil {
			return fmt.Errorf("saving live item %q from %s: %v", sellerSKU, tenant.Tenant().Name, err)
		}
		if err := s.saveTenantInventory(tenant.Tenant().Name, item); err != nil {
			return fmt.Errorf("saving cached item %q from %s: %v", sellerSKU, tenant.Tenant().Name, err)
		}
	}

	return nil
}
