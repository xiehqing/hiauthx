package authorization

import (
	"errors"
	"testing"

	"github.com/xiehqing/hiauthx/db/entity"
)

func TestSystemConfigCategory(t *testing.T) {
	if got := normalizeSystemConfigCategory(""); got != entity.SystemConfigCategorySystem {
		t.Fatalf("empty category should default to system, got %q", got)
	}
	if !validSystemConfigCategory(entity.SystemConfigCategorySystem) || !validSystemConfigCategory(entity.SystemConfigCategorySite) {
		t.Fatal("supported categories were rejected")
	}
	if validSystemConfigCategory("other") {
		t.Fatal("unsupported category was accepted")
	}
	err := validateSystemConfig("site.title", "网站标题", entity.ConfigValueTypeString, "other", entity.ConfigEnabled)
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestFilterVisibleSystemConfigs(t *testing.T) {
	configs := []entity.SystemConfig{
		{Key: entity.SecurityTokenStorageType, Value: "redis"},
		{Key: entity.SecurityTokenRedisConfig, Value: "{}", VisibleWhenKey: entity.SecurityTokenStorageType, VisibleWhenValue: "redis"},
		{Key: "security.token.memory.config", Value: "{}", VisibleWhenKey: entity.SecurityTokenStorageType, VisibleWhenValue: "memory"},
		{Key: "security.login.rsa.public_key", Value: "key", VisibleWhenKey: entity.SecurityLoginEncryptEnable, VisibleWhenValue: "true"},
		{Key: entity.SecurityLoginEncryptEnable, Value: "1"},
	}

	result := filterVisibleSystemConfigs(configs, configs)
	if len(result) != 4 {
		t.Fatalf("expected 4 visible configs, got %d: %+v", len(result), result)
	}

	visible := make(map[string]bool, len(result))
	for _, config := range result {
		visible[config.Key] = true
	}
	if !visible[entity.SecurityTokenRedisConfig] {
		t.Fatal("redis config should be visible when token storage type is redis")
	}
	if visible["security.token.memory.config"] {
		t.Fatal("memory config should be hidden when token storage type is redis")
	}
	if !visible["security.login.rsa.public_key"] {
		t.Fatal("bool-like condition should match 1 and true")
	}
}

func TestConfigValueMatchesBoolLiterals(t *testing.T) {
	if !configValueMatches("1", "true") {
		t.Fatal("1 should match true")
	}
	if configValueMatches("redis", "memory") {
		t.Fatal("different enum values should not match")
	}
}

func TestValidateTokenStorageConfigValue(t *testing.T) {
	if err := validateSystemConfigValue(entity.SecurityTokenStorageType, entity.ConfigValueTypeEnum, "redis"); err != nil {
		t.Fatalf("redis token storage type should be valid: %v", err)
	}
	if err := validateSystemConfigValue(entity.SecurityTokenStorageType, entity.ConfigValueTypeEnum, "mysql"); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("unsupported token storage type should be invalid, got %v", err)
	}
	if err := validateSystemConfigValue(entity.SecurityTokenRedisConfig, entity.ConfigValueTypeJSON, `{"host":"localhost"}`); err != nil {
		t.Fatalf("redis json config should be valid: %v", err)
	}
	if err := validateSystemConfigValue(entity.SecurityTokenRedisConfig, entity.ConfigValueTypeJSON, `{`); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("invalid redis json config should be rejected, got %v", err)
	}
}

func TestValidateSystemConfigOptions(t *testing.T) {
	options := `[{"label":"内存","value":"memory"},{"label":"Redis","value":"redis"}]`
	if err := validateSystemConfigOptions(entity.ConfigValueTypeEnum, options); err != nil {
		t.Fatalf("enum options should be valid: %v", err)
	}
	if err := validateSystemConfigOptions(entity.ConfigValueTypeEnum, `{`); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("invalid enum options should be rejected, got %v", err)
	}
	if err := validateSystemConfigOptions(entity.ConfigValueTypeString, options); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("non-enum options should be rejected, got %v", err)
	}
}

func TestMergeBatchSystemConfigPreservesMetadata(t *testing.T) {
	value := "redis"
	existing := entity.SystemConfig{
		Key:       entity.SecurityTokenStorageType,
		Value:     "memory",
		Name:      "Token 存储类型",
		ValueType: entity.ConfigValueTypeEnum,
		Options:   `[{"label":"内存","value":"memory"},{"label":"Redis","value":"redis"}]`,
		Group:     "token_storage",
		Category:  entity.SystemConfigCategorySystem,
		Enabled:   entity.ConfigEnabled,
	}

	config, err := mergeBatchSystemConfig(existing, BatchSaveSystemConfigItem{Value: &value}, "tester")
	if err != nil {
		t.Fatalf("mergeBatchSystemConfig returned error: %v", err)
	}
	if config.Value != "redis" {
		t.Fatalf("value should be updated, got %q", config.Value)
	}
	if config.Name != existing.Name || config.ValueType != existing.ValueType || config.Options != existing.Options || config.Group != existing.Group {
		t.Fatalf("metadata should be preserved: %+v", config)
	}
	if config.UpdatedBy != "tester" {
		t.Fatalf("operator should be applied, got %q", config.UpdatedBy)
	}
}

func TestBuildNewBatchSystemConfig(t *testing.T) {
	name := "Token 存储类型"
	valueType := entity.ConfigValueTypeEnum
	config, err := buildNewBatchSystemConfig(BatchSaveSystemConfigItem{
		Key:       entity.SecurityTokenStorageType,
		Name:      &name,
		ValueType: &valueType,
	}, "tester")
	if err != nil {
		t.Fatalf("buildNewBatchSystemConfig returned error: %v", err)
	}
	if config.Value != "memory" {
		t.Fatalf("empty token storage type should default to memory, got %q", config.Value)
	}
	if config.CreatedBy != "tester" || config.UpdatedBy != "tester" {
		t.Fatalf("operator should be applied: %+v", config.BaseModel)
	}
}
