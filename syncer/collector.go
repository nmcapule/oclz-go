package syncer

import (
	"fmt"

	"github.com/nmcapule/oclz-go/integrations/models"

	log "github.com/sirupsen/logrus"
)

// CollectAllItems collects and saves fresh item details from each of the
// registered tenants for the syncer.
func (s *Syncer) CollectAllItems() error {
	intentItems, err := s.IntentTenant.CollectAllItems()
	if err != nil {
		return fmt.Errorf("collect all intent items: %v", err)
	}
	intentItemsLookup := make(map[string]struct{})
	for _, item := range intentItems {
		intentItemsLookup[item.SellerSKU] = struct{}{}
	}

	// Collect all items that are not intent items.
	itemsOutsideIntent := make(map[string]*models.Item)
	for _, tenant := range s.nonIntentTenants() {
		items, err := tenant.CollectAllItems()
		if err != nil {
			return fmt.Errorf("collect tenant items for %q: %v", tenant.Tenant().Name, err)
		}
		for _, item := range items {
			item := item
			_, err := s.tenantInventory(tenant.Tenant().Name, item.SellerSKU)
			// If not found, means that this is the first time we detected
			// the item on this tenant.
			if err == models.ErrNotFound {
				log.WithFields(log.Fields{
					"tenant":     tenant.Tenant().Name,
					"seller_sku": item.SellerSKU,
				}).Infof("Recording tenant inventory for the first time")
				// Save fresh copy to the tenant inventory.
				err = s.saveTenantInventory(tenant.Tenant().Name, item)
				if err != nil {
					return fmt.Errorf("save fresh item: %v", err)
				}
			} else if err != nil {
				return fmt.Errorf("retrieving cached item for %s: %v", item.SellerSKU, err)
			}
			if _, ok := intentItemsLookup[item.SellerSKU]; !ok {
				itemsOutsideIntent[item.SellerSKU] = item
			}
		}
	}

	// Save all new items that are not in the intent into the intent.
	for _, item := range itemsOutsideIntent {
		log.WithFields(log.Fields{
			"tenant":     s.IntentTenant.Tenant().Name,
			"seller_sku": item.SellerSKU,
		}).Infof("Recording intent tenant inventory")
		err := s.saveTenantInventory(s.IntentTenant.Tenant().Name, item)
		if err != nil {
			return fmt.Errorf("save tenant items: %v", err)
		}
	}

	return nil
}
