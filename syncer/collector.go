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
		for i, item := range items {
			_, err := s.tenantInventory(tenant.Tenant().Name, item.SellerSKU)
			// If not found, means that this is the first time we detected
			// the item on this tenant.
			if err == models.ErrNotFound {
				log.Infof("recording tenant inventory: %s: %s", tenant.Tenant().Name, item.SellerSKU)
				// Get fresh copy from the tenant.
				fresh, err := tenant.LoadItem(item.SellerSKU)
				if err != nil {
					return fmt.Errorf("get fresh item: %v", err)
				}
				// Save fresh copy to the tenant inventory.
				err = s.saveTenantInventory(tenant.Tenant().Name, fresh)
				if err != nil {
					return fmt.Errorf("save fresh item: %v", err)
				}
			} else if err != nil {
				return fmt.Errorf("retrieving cached item for %s: %v", item.SellerSKU, err)
			}
			if _, ok := intentItemsLookup[item.SellerSKU]; !ok {
				itemsOutsideIntent[item.SellerSKU] = items[i]
			}
		}
	}

	// Save all new items that are not in the intent into the intent.
	for sku, item := range itemsOutsideIntent {
		log.Infof("Recording intent item: %s", sku)
		err := s.saveTenantInventory(s.IntentTenant.Tenant().Name, item)
		if err != nil {
			return fmt.Errorf("save tenant items: %v", err)
		}
	}

	return nil
}
