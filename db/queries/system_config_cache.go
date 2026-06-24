package queries

import (
	"context"
	"strings"
	"sync/atomic"

	"github.com/xiehqing/hiauthx/db/entity"
)

type systemConfigCache struct {
	snapshot atomic.Value
}

func newSystemConfigCache() *systemConfigCache {
	cache := &systemConfigCache{}
	cache.snapshot.Store(map[string]entity.SystemConfig{})
	return cache
}

func (c *systemConfigCache) replace(items []entity.SystemConfig) {
	snapshot := make(map[string]entity.SystemConfig, len(items))
	for _, item := range items {
		key := normalizeConfigCacheKey(item.Key)
		if key == "" {
			continue
		}
		snapshot[key] = item
	}
	c.snapshot.Store(snapshot)
}

func (c *systemConfigCache) get(key string) (entity.SystemConfig, bool) {
	snapshot := c.snapshot.Load().(map[string]entity.SystemConfig)
	item, ok := snapshot[normalizeConfigCacheKey(key)]
	return item, ok
}

func normalizeConfigCacheKey(key string) string {
	return strings.TrimSpace(key)
}

func (q *Queries) RefreshSystemConfigCache(ctx context.Context) error {
	items, err := q.listAllSystemConfigs(ctx)
	if err != nil {
		return err
	}
	q.systemConfigCache.replace(items)
	return nil
}

func (q *Queries) refreshSystemConfigCache(ctx context.Context) error {
	return q.RefreshSystemConfigCache(ctx)
}
