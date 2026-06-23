package entity

import (
	"strings"

	"github.com/xiehqing/infra/pkg/ormx"
)

const (
	ConfigValueTypeString = "string"
	ConfigValueTypeNumber = "number"
	ConfigValueTypeBool   = "bool"
	ConfigValueTypeJSON   = "json"
	ConfigValueTypeEnum   = "enum"

	ConfigDisabled = 0
	ConfigEnabled  = 1

	SystemConfigCategorySystem = "system"
	SystemConfigCategorySite   = "site"
)

const (
	SiteTitle     = "site.title"
	SiteLogo      = "site.logo"
	SiteFavicon   = "site.favicon"
	SiteCopyright = "site.copyright"

	SecurityPasswordMinLength = "security.password.min_length"

	SecurityLoginEncryptEnable    = "security.login.encrypt.enabled"
	SecurityLoginRSAPublicKey     = "security.login.rsa.public_key"
	SecurityLoginRSAPrivateKey    = "security.login.rsa.private_key"
	SecurityLoginMaxAttempts      = "security.login.max_attempts"
	SecurityLoginLockedMinutes    = "security.login.locked_minutes"
	SecurityLoginConcurrentEnable = "security.login.concurrent.enabled"

	SecurityTokenExpireMinutes = "security.token.expire_minutes"
	SecurityTokenJwtSecretKey  = "security.token.jwt_secret_key"
	SecurityTokenStorage       = "security.token.storage"
	SecurityTokenStorageType   = "security.token.storage.type"
	SecurityTokenRedisConfig   = "security.token.storage.redis"
)

type SystemConfig struct {
	ormx.BaseModel
	Key              string `json:"key" gorm:"column:config_key;type:varchar(128);not null;uniqueIndex;comment:'配置键'"`
	Value            string `json:"value" gorm:"column:config_value;type:text;comment:'配置值'"`
	Name             string `json:"name" gorm:"type:varchar(128);not null;comment:'配置名称'"`
	ValueType        string `json:"valueType" gorm:"type:varchar(32);not null;default:'string';comment:'配置值类型'"`
	Group            string `json:"group" gorm:"column:config_group;type:varchar(64);not null;default:'default';index;comment:'配置分组'"`
	Category         string `json:"category" gorm:"type:varchar(32);not null;default:'system';index;comment:'配置类别：system系统设置，site网站配置'"`
	VisibleWhenKey   string `json:"visibleWhenKey" gorm:"type:varchar(128);comment:'展示条件配置键'"`
	VisibleWhenValue string `json:"visibleWhenValue" gorm:"type:varchar(64);comment:'展示条件配置值'"`
	Description      string `json:"description" gorm:"type:varchar(255);comment:'配置描述'"`
	Enabled          int    `json:"enabled" gorm:"type:int(11);not null;default:1;index;comment:'是否启用：0禁用，1启用'"`
	Sort             int    `json:"sort" gorm:"type:int(11);not null;default:0;comment:'排序'"`
}

func (s *SystemConfig) TableName() string {
	return "system_config"
}

func ConfigBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}
