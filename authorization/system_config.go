package authorization

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/hiauthx/db/queries"
	"github.com/xiehqing/infra/pkg/ormx"
	"regexp"
	"strings"
)

var configKeyPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.:-]*$`)

type CreateSystemConfigRequest struct {
	Key              string `json:"key"`
	Value            string `json:"value"`
	Name             string `json:"name"`
	ValueType        string `json:"valueType"`
	Options          string `json:"options"`
	Group            string `json:"group"`
	Category         string `json:"category"`
	VisibleWhenKey   string `json:"visibleWhenKey"`
	VisibleWhenValue string `json:"visibleWhenValue"`
	Description      string `json:"description"`
	Enabled          *int   `json:"enabled"`
	Sort             int    `json:"sort"`
	Operator         string `json:"operator"`
}

type UpdateSystemConfigRequest struct {
	ID               int64  `json:"id"`
	Key              string `json:"key"`
	Value            string `json:"value"`
	Name             string `json:"name"`
	ValueType        string `json:"valueType"`
	Options          string `json:"options"`
	Group            string `json:"group"`
	Category         string `json:"category"`
	VisibleWhenKey   string `json:"visibleWhenKey"`
	VisibleWhenValue string `json:"visibleWhenValue"`
	Description      string `json:"description"`
	Enabled          int    `json:"enabled"`
	Sort             int    `json:"sort"`
	Operator         string `json:"operator"`
}

type BatchSaveSystemConfigsRequest struct {
	Items    []BatchSaveSystemConfigItem `json:"items"`
	Operator string                      `json:"operator"`
}

type BatchSaveSystemConfigItem struct {
	ID               int64   `json:"id"`
	Key              string  `json:"key"`
	Value            *string `json:"value"`
	Name             *string `json:"name"`
	ValueType        *string `json:"valueType"`
	Options          *string `json:"options"`
	Group            *string `json:"group"`
	Category         *string `json:"category"`
	VisibleWhenKey   *string `json:"visibleWhenKey"`
	VisibleWhenValue *string `json:"visibleWhenValue"`
	Description      *string `json:"description"`
	Enabled          *int    `json:"enabled"`
	Sort             *int    `json:"sort"`
	Operator         string  `json:"operator"`
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
	key := normalizeString(req.Key)
	value := normalizeSystemConfigValue(key, req.Value)

	if err := validateSystemConfig(key, req.Name, valueType, category, enabled); err != nil {
		return nil, err
	}
	if err := validateSystemConfigValue(key, valueType, value); err != nil {
		return nil, err
	}
	options := normalizeString(req.Options)
	if err := validateSystemConfigOptions(valueType, options); err != nil {
		return nil, err
	}

	config := &entity.SystemConfig{
		BaseModel: ormx.BaseModel{
			CreatedBy: normalizeString(req.Operator),
			UpdatedBy: normalizeString(req.Operator),
		},
		Key:              key,
		Value:            value,
		Name:             normalizeString(req.Name),
		ValueType:        valueType,
		Options:          options,
		Group:            group,
		Category:         category,
		VisibleWhenKey:   normalizeString(req.VisibleWhenKey),
		VisibleWhenValue: normalizeString(req.VisibleWhenValue),
		Description:      normalizeString(req.Description),
		Enabled:          enabled,
		Sort:             req.Sort,
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
	key := normalizeString(req.Key)
	value := normalizeSystemConfigValue(key, req.Value)
	if err := validateSystemConfig(key, req.Name, valueType, category, req.Enabled); err != nil {
		return nil, err
	}
	if err := validateSystemConfigValue(key, valueType, value); err != nil {
		return nil, err
	}
	options := normalizeString(req.Options)
	if err := validateSystemConfigOptions(valueType, options); err != nil {
		return nil, err
	}

	config := &entity.SystemConfig{
		BaseModel: ormx.BaseModel{
			ID:        req.ID,
			UpdatedBy: normalizeString(req.Operator),
		},
		Key:              key,
		Value:            value,
		Name:             normalizeString(req.Name),
		ValueType:        valueType,
		Options:          options,
		Group:            group,
		Category:         category,
		VisibleWhenKey:   normalizeString(req.VisibleWhenKey),
		VisibleWhenValue: normalizeString(req.VisibleWhenValue),
		Description:      normalizeString(req.Description),
		Enabled:          req.Enabled,
		Sort:             req.Sort,
	}

	if err := as.queries.UpdateSystemConfig(ctx, config); err != nil {
		return nil, normalizeDBError(err)
	}
	return as.GetSystemConfig(ctx, req.ID)
}

func (as *Service) BatchSaveSystemConfigs(ctx context.Context, req BatchSaveSystemConfigsRequest) ([]entity.SystemConfig, error) {
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("%w: 批量保存配置不能为空", ErrInvalidArgument)
	}

	configs := make([]*entity.SystemConfig, 0, len(req.Items))
	seen := make(map[string]struct{}, len(req.Items))
	for i, item := range req.Items {
		config, err := as.buildBatchSaveSystemConfig(ctx, item, req.Operator)
		if err != nil {
			return nil, fmt.Errorf("第 %d 项: %w", i+1, err)
		}
		key := config.Key
		if config.ID > 0 {
			key = fmt.Sprintf("id:%d", config.ID)
		}
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("%w: 批量保存配置存在重复项 %s", ErrInvalidArgument, key)
		}
		seen[key] = struct{}{}
		configs = append(configs, config)
	}

	if err := as.queries.SaveSystemConfigs(ctx, configs); err != nil {
		return nil, normalizeDBError(err)
	}

	result := make([]entity.SystemConfig, 0, len(configs))
	for _, config := range configs {
		saved, err := as.queries.GetSystemConfig(ctx, config.ID)
		if err != nil {
			return nil, err
		}
		if saved != nil {
			result = append(result, *saved)
		}
	}
	return result, nil
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
	configs, err := as.queries.ListEnabledSystemConfigs(ctx, normalizeString(group), category)
	if err != nil {
		return nil, err
	}
	if !hasVisibleCondition(configs) {
		return configs, nil
	}

	allConfigs := configs
	if normalizeString(group) != "" || category != "" {
		allConfigs, err = as.queries.ListEnabledSystemConfigs(ctx, "", "")
		if err != nil {
			return nil, err
		}
	}
	return filterVisibleSystemConfigs(configs, allConfigs), nil
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

func normalizeSystemConfigValue(key, value string) string {
	if key == entity.SecurityTokenStorageType {
		value = strings.ToLower(normalizeString(value))
		if value == "" {
			return "memory"
		}
	}
	return value
}

func (as *Service) buildBatchSaveSystemConfig(ctx context.Context, item BatchSaveSystemConfigItem, operator string) (*entity.SystemConfig, error) {
	existing, err := as.findBatchSaveSystemConfig(ctx, item)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		return buildNewBatchSystemConfig(item, operator)
	}
	return mergeBatchSystemConfig(*existing, item, operator)
}

func (as *Service) findBatchSaveSystemConfig(ctx context.Context, item BatchSaveSystemConfigItem) (*entity.SystemConfig, error) {
	if item.ID > 0 {
		config, err := as.GetSystemConfig(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		return config, nil
	}

	key := normalizeString(item.Key)
	if key == "" {
		return nil, nil
	}
	return as.queries.GetSystemConfigByKey(ctx, key)
}

func buildNewBatchSystemConfig(item BatchSaveSystemConfigItem, operator string) (*entity.SystemConfig, error) {
	key := normalizeString(item.Key)
	if key == "" {
		return nil, fmt.Errorf("%w: 配置键不能为空", ErrInvalidArgument)
	}
	name := stringValue(item.Name, "")
	valueType := normalizeConfigValueType(stringValue(item.ValueType, ""))
	options := normalizeString(stringValue(item.Options, ""))
	group := normalizeConfigGroup(stringValue(item.Group, ""))
	category := normalizeSystemConfigCategory(stringValue(item.Category, ""))
	enabled := intValue(item.Enabled, entity.ConfigEnabled)
	value := normalizeSystemConfigValue(key, stringValue(item.Value, ""))

	if err := validateSystemConfig(key, name, valueType, category, enabled); err != nil {
		return nil, err
	}
	if err := validateSystemConfigValue(key, valueType, value); err != nil {
		return nil, err
	}
	if err := validateSystemConfigOptions(valueType, options); err != nil {
		return nil, err
	}

	operator = batchItemOperator(item.Operator, operator)
	return &entity.SystemConfig{
		BaseModel: ormx.BaseModel{
			CreatedBy: operator,
			UpdatedBy: operator,
		},
		Key:              key,
		Value:            value,
		Name:             normalizeString(name),
		ValueType:        valueType,
		Options:          options,
		Group:            group,
		Category:         category,
		VisibleWhenKey:   normalizeString(stringValue(item.VisibleWhenKey, "")),
		VisibleWhenValue: normalizeString(stringValue(item.VisibleWhenValue, "")),
		Description:      normalizeString(stringValue(item.Description, "")),
		Enabled:          enabled,
		Sort:             intValue(item.Sort, 0),
	}, nil
}

func mergeBatchSystemConfig(existing entity.SystemConfig, item BatchSaveSystemConfigItem, operator string) (*entity.SystemConfig, error) {
	config := existing
	if item.Key != "" {
		config.Key = normalizeString(item.Key)
	}
	if item.Value != nil {
		config.Value = *item.Value
	}
	if item.Name != nil {
		config.Name = normalizeString(*item.Name)
	}
	if item.ValueType != nil {
		config.ValueType = normalizeConfigValueType(*item.ValueType)
	}
	if item.Options != nil {
		config.Options = normalizeString(*item.Options)
	}
	if item.Group != nil {
		config.Group = normalizeConfigGroup(*item.Group)
	}
	if item.Category != nil {
		config.Category = normalizeSystemConfigCategory(*item.Category)
	}
	if item.VisibleWhenKey != nil {
		config.VisibleWhenKey = normalizeString(*item.VisibleWhenKey)
	}
	if item.VisibleWhenValue != nil {
		config.VisibleWhenValue = normalizeString(*item.VisibleWhenValue)
	}
	if item.Description != nil {
		config.Description = normalizeString(*item.Description)
	}
	if item.Enabled != nil {
		config.Enabled = *item.Enabled
	}
	if item.Sort != nil {
		config.Sort = *item.Sort
	}
	config.Value = normalizeSystemConfigValue(config.Key, config.Value)
	config.UpdatedBy = batchItemOperator(item.Operator, operator)

	if err := validateSystemConfig(config.Key, config.Name, config.ValueType, config.Category, config.Enabled); err != nil {
		return nil, err
	}
	if err := validateSystemConfigValue(config.Key, config.ValueType, config.Value); err != nil {
		return nil, err
	}
	if err := validateSystemConfigOptions(config.ValueType, config.Options); err != nil {
		return nil, err
	}
	return &config, nil
}

func batchItemOperator(itemOperator, batchOperator string) string {
	operator := normalizeString(itemOperator)
	if operator != "" {
		return operator
	}
	return normalizeString(batchOperator)
}

func stringValue(value *string, defaultValue string) string {
	if value == nil {
		return defaultValue
	}
	return *value
}

func intValue(value *int, defaultValue int) int {
	if value == nil {
		return defaultValue
	}
	return *value
}

func validConfigValueType(valueType string) bool {
	switch valueType {
	case entity.ConfigValueTypeString,
		entity.ConfigValueTypeNumber,
		entity.ConfigValueTypeBool,
		entity.ConfigValueTypeJSON,
		entity.ConfigValueTypeEnum:
		return true
	default:
		return false
	}
}

func validateSystemConfigValue(key, valueType, value string) error {
	switch key {
	case entity.SecurityTokenStorageType:
		if valueType != entity.ConfigValueTypeEnum {
			return fmt.Errorf("%w: Token 存储类型必须使用 enum 类型", ErrInvalidArgument)
		}
		if !validTokenStorageType(value) {
			return fmt.Errorf("%w: Token 存储类型只支持 memory 或 redis", ErrInvalidArgument)
		}
	case entity.SecurityTokenRedisConfig:
		if valueType != entity.ConfigValueTypeJSON {
			return fmt.Errorf("%w: Token Redis 配置必须使用 json 类型", ErrInvalidArgument)
		}
		if normalizeString(value) != "" && !json.Valid([]byte(value)) {
			return fmt.Errorf("%w: Token Redis 配置不是合法 JSON", ErrInvalidArgument)
		}
	}
	return nil
}

func validateSystemConfigOptions(valueType, options string) error {
	options = normalizeString(options)
	if options == "" {
		return nil
	}
	if valueType != entity.ConfigValueTypeEnum {
		return fmt.Errorf("%w: 只有 enum 类型支持配置选项", ErrInvalidArgument)
	}
	if !json.Valid([]byte(options)) {
		return fmt.Errorf("%w: 配置选项不是合法 JSON", ErrInvalidArgument)
	}
	return nil
}

func validTokenStorageType(value string) bool {
	switch strings.ToLower(normalizeString(value)) {
	case "memory", "redis":
		return true
	default:
		return false
	}
}

func filterVisibleSystemConfigs(configs, conditionConfigs []entity.SystemConfig) []entity.SystemConfig {
	values := make(map[string]string, len(conditionConfigs))
	for _, config := range conditionConfigs {
		values[config.Key] = config.Value
	}

	result := make([]entity.SystemConfig, 0, len(configs))
	for _, config := range configs {
		if systemConfigVisible(config, values) {
			result = append(result, config)
		}
	}
	return result
}

func hasVisibleCondition(configs []entity.SystemConfig) bool {
	for _, config := range configs {
		if normalizeString(config.VisibleWhenKey) != "" {
			return true
		}
	}
	return false
}

func systemConfigVisible(config entity.SystemConfig, values map[string]string) bool {
	key := normalizeString(config.VisibleWhenKey)
	if key == "" {
		return true
	}
	expected := normalizeString(config.VisibleWhenValue)
	actual, ok := values[key]
	if !ok {
		return false
	}
	return configValueMatches(actual, expected)
}

func configValueMatches(actual, expected string) bool {
	actual = normalizeString(actual)
	expected = normalizeString(expected)
	if isBoolLiteral(actual) && isBoolLiteral(expected) {
		return entity.ConfigBool(actual) == entity.ConfigBool(expected)
	}
	return actual == expected
}

func isBoolLiteral(value string) bool {
	switch strings.ToLower(value) {
	case "1", "0", "true", "false", "yes", "no", "on", "off", "enabled", "disabled":
		return true
	default:
		return false
	}
}
