package processing

import "github.com/Nox1KCL/Arbitrage/internal/database/models"

func NewTestSubscriptionStore(maps *models.CachedMaps) *SubscriptionStore {
	store := &SubscriptionStore{}
	store.ptr.Store(maps)
	return store
}
