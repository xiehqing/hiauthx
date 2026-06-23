package queries

import (
	"context"
	"strings"

	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/infra/pkg/ormx"
	"gorm.io/gorm"
)

type APIListFilter struct {
	ormx.Pagination
	Method string `json:"method" form:"method"`
	Module string `json:"module" form:"module"`
	Action string `json:"action" form:"action"`
	Status *int   `json:"status" form:"status"`
}

func (q *Queries) CreateAPI(ctx context.Context, item *entity.API) error {
	return q.db.WithContext(ctx).Create(item).Error
}

func (q *Queries) UpdateAPI(ctx context.Context, item *entity.API) error {
	result := q.db.WithContext(ctx).Model(&entity.API{}).Where("id = ?", item.ID).
		Select("Name", "Method", "Path", "Module", "Action", "Description", "ResourceType", "Status", "Sort", "UpdatedBy").Updates(item)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (q *Queries) GetAPI(ctx context.Context, id int64) (*entity.API, error) {
	var item entity.API
	err := q.db.WithContext(ctx).First(&item, "id = ?", id).Error
	if err != nil {
		return nil, ormx.NotFoundAsNil(err)
	}
	return &item, nil
}

func (q *Queries) ListEnabledAPIs(ctx context.Context) ([]entity.API, error) {
	var items []entity.API
	err := q.db.WithContext(ctx).
		Where("status = ?", entity.APIStatusEnabled).
		Order("sort asc, id asc").
		Find(&items).Error
	return items, err
}

func (q *Queries) DeleteAPI(ctx context.Context, id int64) error {
	result := q.db.WithContext(ctx).Delete(&entity.API{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (q *Queries) ListAPIs(ctx context.Context, filter APIListFilter) (ormx.PageResult[entity.API], error) {
	db := q.db.WithContext(ctx).Model(&entity.API{})
	if ormx.KeywordPresent(filter.Keyword) {
		keyword := ormx.LikeKeyword(filter.Keyword)
		db = db.Where("name LIKE ? OR path LIKE ? OR description LIKE ?", keyword, keyword, keyword)
	}
	if filter.Method != "" {
		db = db.Where("method = ?", strings.ToUpper(filter.Method))
	}
	if filter.Module != "" {
		db = db.Where("module = ?", filter.Module)
	}
	if filter.Action != "" {
		db = db.Where("action = ?", filter.Action)
	}
	if filter.Status != nil {
		db = db.Where("status = ?", *filter.Status)
	}
	return ormx.Paginate[entity.API](db, filter.Pagination, map[string]string{
		"id": "id", "name": "name", "method": "method", "path": "path", "module": "module",
		"action": "action", "status": "status", "sort": "sort", "createdAt": "created_at", "updatedAt": "updated_at",
	})
}
