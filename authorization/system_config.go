package authorization

import (
	"context"
	"fmt"
	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/hiauthx/db/queries"
	"github.com/xiehqing/infra/pkg/ormx"
	"regexp"
)

var configKeyPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.:-]*$`)

type CreateSystemConfigRequest struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Name        string `json:"name"`
	ValueType   string `json:"valueType"`
	Group       string `json:"group"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Enabled     *int   `json:"enabled"`
	Sort        int    `json:"sort"`
	Operator    string `json:"operator"`
}

type UpdateSystemConfigRequest struct {
	ID          int64  `json:"id"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Name        string `json:"name"`
	ValueType   string `json:"valueType"`
	Group       string `json:"group"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Enabled     int    `json:"enabled"`
	Sort        int    `json:"sort"`
	Operator    string `json:"operator"`
}

type ListSystemConfigsRequest struct {
	queries.SystemConfigListFilter
}

func (as *Service) CreateSystemConfig(ctx context.Context, req CreateSystemConfigRequest) (*entity.SystemConfig, error) {
	enabled := entity.ConfigEnabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	valueType := normalizeConfigValueType(req.ValueType)
	group := normalizeConfigGroup(req.Group)
	category := normalizeSystemConfigCategory(req.Category)

	if err := validateSystemConfig(req.Key, req.Name, valueType, category, enabled); err != nil {
		return nil, err
	}

	config := &entity.SystemConfig{
		BaseModel: ormx.BaseModel{
			CreatedBy: normalizeString(req.Operator),
			UpdatedBy: normalizeString(req.Operator),
		},
		Key:         normalizeString(req.Key),
		Value:       req.Value,
		Name:        normalizeString(req.Name),
		ValueType:   valueType,
		Group:       group,
		Category:    category,
		Description: normalizeString(req.Description),
		Enabled:     enabled,
		Sort:        req.Sort,
	}

	if err := as.queries.CreateSystemConfig(ctx, config); err != nil {
		return nil, normalizeDBError(err)
	}
	return as.queries.GetSystemConfig(ctx, config.ID)
}

func (as *Service) UpdateSystemConfig(ctx context.Context, req UpdateSystemConfigRequest) (*entity.SystemConfig, error) {
	if req.ID <= 0 {
		return nil, ErrInvalidID
	}

	valueType := normalizeConfigValueType(req.ValueType)
	group := normalizeConfigGroup(req.Group)
	category := normalizeSystemConfigCategory(req.Category)
	if err := validateSystemConfig(req.Key, req.Name, valueType, category, req.Enabled); err != nil {
		return nil, err
	}

	config := &entity.SystemConfig{
		BaseModel: ormx.BaseModel{
			ID:        req.ID,
			UpdatedBy: normalizeString(req.Operator),
		},
		Key:         normalizeString(req.Key),
		Value:       req.Value,
		Name:        normalizeString(req.Name),
		ValueType:   valueType,
		Group:       group,
		Category:    category,
		Description: normalizeString(req.Description),
		Enabled:     req.Enabled,
		Sort:        req.Sort,
	}

	if err := as.queries.UpdateSystemConfig(ctx, config); err != nil {
		return nil, normalizeDBError(err)
	}
	return as.GetSystemConfig(ctx, req.ID)
}

func (as *Service) GetSystemConfig(ctx context.Context, id int64) (*entity.SystemConfig, error) {
	if id <= 0 {
		return nil, ErrInvalidID
	}
	config, err := as.queries.GetSystemConfig(ctx, id)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, ErrNotFound
	}
	return config, nil
}

func (as *Service) GetSystemConfigByKey(ctx context.Context, key string) (*entity.SystemConfig, error) {
	key = normalizeString(key)
	if key == "" {
		return nil, fmt.Errorf("%w: 配置键不能为空", ErrInvalidArgument)
	}
	config, err := as.queries.GetSystemConfigByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, ErrNotFound
	}
	return config, nil
}

func (as *Service) ListSystemConfigs(ctx context.Context, req ListSystemConfigsRequest) (ormx.PageResult[entity.SystemConfig], error) {
	req.Group = normalizeString(req.Group)
	req.Category = normalizeString(req.Category)
	if req.Category != "" && !validSystemConfigCategory(req.Category) {
		return ormx.PageResult[entity.SystemConfig]{}, fmt.Errorf("%w: 配置类别不合法", ErrInvalidArgument)
	}
	return as.queries.ListSystemConfigs(ctx, req.SystemConfigListFilter)
}

func (as *Service) ListEnabledSystemConfigs(ctx context.Context, group, category string) ([]entity.SystemConfig, error) {
	category = normalizeString(category)
	if category != "" && !validSystemConfigCategory(category) {
		return nil, fmt.Errorf("%w: 配置类别不合法", ErrInvalidArgument)
	}
	return as.queries.ListEnabledSystemConfigs(ctx, normalizeString(group), category)
}

func (as *Service) GetEnabledSystemConfigMap(ctx context.Context, group, category string) (map[string]string, error) {
	configs, err := as.ListEnabledSystemConfigs(ctx, group, category)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(configs))
	for _, config := range configs {
		result[config.Key] = config.Value
	}
	return result, nil
}

func (as *Service) DeleteSystemConfig(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidID
	}
	return normalizeDBError(as.queries.DeleteSystemConfig(ctx, id))
}

func validateSystemConfig(key, name, valueType, category string, enabled int) error {
	key = normalizeString(key)
	if key == "" || !configKeyPattern.MatchString(key) {
		return fmt.Errorf("%w: 配置键不合法", ErrInvalidArgument)
	}
	if !required(name) {
		return fmt.Errorf("%w: 配置名称不能为空", ErrInvalidArgument)
	}
	if !validConfigValueType(valueType) {
		return fmt.Errorf("%w: 配置值类型不合法", ErrInvalidArgument)
	}
	if !validSystemConfigCategory(category) {
		return fmt.Errorf("%w: 配置类别不合法", ErrInvalidArgument)
	}
	if enabled != entity.ConfigDisabled && enabled != entity.ConfigEnabled {
		return fmt.Errorf("%w: 启用状态不合法", ErrInvalidArgument)
	}
	return nil
}

func normalizeSystemConfigCategory(category string) string {
	category = normalizeString(category)
	if category == "" {
		return entity.SystemConfigCategorySystem
	}
	return category
}

func validSystemConfigCategory(category string) bool {
	return category == entity.SystemConfigCategorySystem || category == entity.SystemConfigCategorySite
}

func normalizeConfigValueType(valueType string) string {
	valueType = normalizeString(valueType)
	if valueType == "" {
		return entity.ConfigValueTypeString
	}
	return valueType
}

func normalizeConfigGroup(group string) string {
	group = normalizeString(group)
	if group == "" {
		return "default"
	}
	return group
}

func validConfigValueType(valueType string) bool {
	switch valueType {
	case entity.ConfigValueTypeString,
		entity.ConfigValueTypeNumber,
		entity.ConfigValueTypeBool,
		entity.ConfigValueTypeJSON:
		return true
	default:
		return false
	}
}
