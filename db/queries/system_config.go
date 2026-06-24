package queries

import (
	"context"
	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/infra/pkg/ormx"

	"gorm.io/gorm"
)

type SystemConfigListFilter struct {
	ormx.Pagination
	Group    string `json:"group" form:"group"`
	Category string `json:"category" form:"category"`
	Enabled  *int   `json:"enabled" form:"enabled"`
}

func (q *Queries) CreateSystemConfig(ctx context.Context, config *entity.SystemConfig) error {
	if err := q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Create(config).Error
	}); err != nil {
		return err
	}
	return q.refreshSystemConfigCache(ctx)
}

func (q *Queries) UpdateSystemConfig(ctx context.Context, config *entity.SystemConfig) error {
	if err := q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return updateSystemConfigTx(tx, config)
	}); err != nil {
		return err
	}
	return q.refreshSystemConfigCache(ctx)
}

func (q *Queries) SaveSystemConfigs(ctx context.Context, configs []*entity.SystemConfig) error {
	if err := q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, config := range configs {
			if config.ID <= 0 {
				if err := tx.Create(config).Error; err != nil {
					return err
				}
				continue
			}
			if err := updateSystemConfigTx(tx, config); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return q.refreshSystemConfigCache(ctx)
}

func updateSystemConfigTx(tx *gorm.DB, config *entity.SystemConfig) error {
	return tx.
		Model(&entity.SystemConfig{}).
		Where("id = ?", config.ID).
		Select("Key", "Value", "Name", "ValueType", "Options", "Group", "Category", "VisibleWhenKey", "VisibleWhenValue", "Description", "Enabled", "Sort", "UpdatedBy").
		Updates(config).
		Error
}

func (q *Queries) GetSystemConfig(ctx context.Context, id int64) (*entity.SystemConfig, error) {
	var config entity.SystemConfig
	err := q.db.WithContext(ctx).First(&config, "id = ?", id).Error
	if err != nil {
		return nil, ormx.NotFoundAsNil(err)
	}
	return &config, nil
}

func (q *Queries) GetSystemConfigByKey(ctx context.Context, key string) (*entity.SystemConfig, error) {
	if config, ok := q.systemConfigCache.get(key); ok {
		return &config, nil
	}
	var config entity.SystemConfig
	err := q.db.WithContext(ctx).First(&config, "config_key = ?", key).Error
	if err != nil {
		return nil, ormx.NotFoundAsNil(err)
	}
	_ = q.refreshSystemConfigCache(ctx)
	return &config, nil
}

func (q *Queries) DeleteSystemConfig(ctx context.Context, id int64) error {
	if err := q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		config := entity.SystemConfig{BaseModel: ormx.BaseModel{ID: id}}
		result := tx.Delete(&config)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	}); err != nil {
		return err
	}
	return q.refreshSystemConfigCache(ctx)
}

func (q *Queries) ListSystemConfigs(ctx context.Context, filter SystemConfigListFilter) (ormx.PageResult[entity.SystemConfig], error) {
	db := q.db.WithContext(ctx).Model(&entity.SystemConfig{})
	if ormx.KeywordPresent(filter.Keyword) {
		keyword := ormx.LikeKeyword(filter.Keyword)
		db = db.Where("config_key LIKE ? OR name LIKE ? OR description LIKE ?", keyword, keyword, keyword)
	}
	if filter.Group != "" {
		db = db.Where("config_group = ?", filter.Group)
	}
	if filter.Category != "" {
		db = db.Where("category = ?", filter.Category)
	}
	if filter.Enabled != nil {
		db = db.Where("enabled = ?", *filter.Enabled)
	}

	return ormx.Paginate[entity.SystemConfig](db, filter.Pagination, map[string]string{
		"id":             "id",
		"key":            "config_key",
		"name":           "name",
		"group":          "config_group",
		"category":       "category",
		"visibleWhenKey": "visible_when_key",
		"enabled":        "enabled",
		"sort":           "sort",
		"createdAt":      "created_at",
		"updatedAt":      "updated_at",
	})
}

func (q *Queries) ListEnabledSystemConfigs(ctx context.Context, group, category string) ([]entity.SystemConfig, error) {
	db := q.db.WithContext(ctx).Where("enabled = ?", entity.ConfigEnabled)
	if group != "" {
		db = db.Where("config_group = ?", group)
	}
	if category != "" {
		db = db.Where("category = ?", category)
	}

	var configs []entity.SystemConfig
	err := db.Order("config_group asc, sort asc, id asc").Find(&configs).Error
	return configs, err
}

func (q *Queries) GetSystemConfigValueByKey(ctx context.Context, key string) (string, error) {
	config, ok := q.systemConfigCache.get(key)
	if !ok {
		item, err := q.GetSystemConfigByKey(ctx, key)
		if err != nil || item == nil {
			return "", err
		}
		return item.Value, nil
	}
	return config.Value, nil
}

func (q *Queries) GetSystemConfigBoolValueByKey(ctx context.Context, key string, defaultValue bool) bool {
	configValue, err := q.GetSystemConfigValueByKey(ctx, key)
	if err != nil {
		return defaultValue
	}
	return entity.ConfigBool(configValue)
}

func (q *Queries) listAllSystemConfigs(ctx context.Context) ([]entity.SystemConfig, error) {
	var configs []entity.SystemConfig
	err := q.db.WithContext(ctx).Find(&configs).Error
	return configs, err
}
