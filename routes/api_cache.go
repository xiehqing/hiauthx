package routes

import (
	"context"
	"strings"
	"sync/atomic"

	"github.com/xiehqing/hiauthx/db/entity"
)

// apiMetadataCache is an immutable route metadata snapshot. Requests only read
// the current snapshot, while refreshes build and atomically swap a new one.
type apiMetadataCache struct {
	snapshot atomic.Value
}

func newAPIMetadataCache() *apiMetadataCache {
	cache := &apiMetadataCache{}
	cache.snapshot.Store(map[string]entity.API{})
	return cache
}

func apiMetadataKey(method, path string) string {
	return strings.ToUpper(strings.TrimSpace(method)) + "\x00" + strings.TrimSpace(path)
}

func (c *apiMetadataCache) replace(items []entity.API) {
	snapshot := make(map[string]entity.API, len(items))
	for _, item := range items {
		if item.Status == entity.APIStatusEnabled {
			snapshot[apiMetadataKey(item.Method, item.Path)] = item
		}
	}
	c.snapshot.Store(snapshot)
}

func (c *apiMetadataCache) get(method, path string) (entity.API, bool) {
	snapshot := c.snapshot.Load().(map[string]entity.API)
	item, ok := snapshot[apiMetadataKey(method, path)]
	return item, ok
}

// RefreshAPICache rebuilds the complete enabled API metadata snapshot.
func (r *Router) RefreshAPICache(ctx context.Context) error {
	items, err := r.q.ListEnabledAPIs(ctx)
	if err != nil {
		return err
	}
	r.apiCache.replace(items)
	return nil
}
