package queries

import (
	"context"
	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/infra/pkg/ormx"

	"gorm.io/gorm"
)

type SystemConfigListFilter struct {
	ormx.Pagination
	Group   string `json:"group" form:"group"`
	Enabled *int   `json:"enabled" form:"enabled"`
}

func (q *Queries) CreateSystemConfig(ctx context.Context, config *entity.SystemConfig) error {
	return q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Create(config).Error
	})
}

func (q *Queries) UpdateSystemConfig(ctx context.Context, config *entity.SystemConfig) error {
	return q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.
			Model(&entity.SystemConfig{}).
			Where("id = ?", config.ID).
			Select("Key", "Value", "Name", "ValueType", "Group", "Description", "Enabled", "Sort", "UpdatedBy").
			Updates(config).
			Error
	})
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
	var config entity.SystemConfig
	err := q.db.WithContext(ctx).First(&config, "config_key = ?", key).Error
	if err != nil {
		return nil, ormx.NotFoundAsNil(err)
	}
	return &config, nil
}

func (q *Queries) DeleteSystemConfig(ctx context.Context, id int64) error {
	return q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		config := entity.SystemConfig{BaseModel: ormx.BaseModel{ID: id}}
		result := tx.Delete(&config)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
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
	if filter.Enabled != nil {
		db = db.Where("enabled = ?", *filter.Enabled)
	}

	return ormx.Paginate[entity.SystemConfig](db, filter.Pagination, map[string]string{
		"id":        "id",
		"key":       "config_key",
		"name":      "name",
		"group":     "config_group",
		"enabled":   "enabled",
		"sort":      "sort",
		"createdAt": "created_at",
		"updatedAt": "updated_at",
	})
}

func (q *Queries) ListEnabledSystemConfigs(ctx context.Context, group string) ([]entity.SystemConfig, error) {
	db := q.db.WithContext(ctx).Where("enabled = ?", entity.ConfigEnabled)
	if group != "" {
		db = db.Where("config_group = ?", group)
	}

	var configs []entity.SystemConfig
	err := db.Order("config_group asc, sort asc, id asc").Find(&configs).Error
	return configs, err
}

func (q *Queries) GetSystemConfigValueByKey(ctx context.Context, key string) (string, error) {
	var config entity.SystemConfig
	err := q.db.WithContext(ctx).First(&config, "config_key = ?", key).Error
	if err != nil {
		return "", ormx.NotFoundAsNil(err)
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
