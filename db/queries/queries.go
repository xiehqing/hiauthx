package queries

import (
	"context"
	"github.com/xiehqing/hiauthx/audit"
	"github.com/xiehqing/hiauthx/db/entity"
	"gorm.io/gorm"
)

type Queries struct {
	db    *gorm.DB
	audit *audit.Queries
}

func New(db *gorm.DB, audit *audit.Queries) *Queries {
	q := &Queries{
		db:    db,
		audit: audit,
	}
	return q
}

func (q *Queries) DB() *gorm.DB {
	return q.db
}

func (q *Queries) AutoMigrate(ctx context.Context) error {
	return q.db.WithContext(ctx).AutoMigrate(
		&entity.User{},
		&entity.Role{},
		&entity.Department{},
		&entity.Menu{},
		&entity.SystemConfig{},
		&entity.API{},
		&entity.AuditLog{},
		&entity.AuditChange{},
	)
}

func normalizeDataIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}
