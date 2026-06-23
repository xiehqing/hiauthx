package entity

import "github.com/xiehqing/infra/pkg/ormx"

const (
	APIStatusDisabled = 0
	APIStatusEnabled  = 1
)

// API stores the audit metadata associated with one HTTP route.
// Path uses the registered route template, for example /api/v1/users/:id.
type API struct {
	ormx.BaseModel
	Name         string `json:"name" gorm:"type:varchar(128);not null;comment:'API name'"`
	Method       string `json:"method" gorm:"type:varchar(16);not null;uniqueIndex:uk_api_method_path;comment:'HTTP method'"`
	Path         string `json:"path" gorm:"type:varchar(255);not null;uniqueIndex:uk_api_method_path;comment:'route template'"`
	Module       string `json:"module" gorm:"type:varchar(64);not null;index;comment:'audit module'"`
	Action       string `json:"action" gorm:"type:varchar(64);not null;index;comment:'audit action'"`
	Description  string `json:"description" gorm:"type:varchar(255);comment:'audit description'"`
	ResourceType string `json:"resourceType" gorm:"type:varchar(64);not null;default:'request';index;comment:'resource type'"`
	Status       int    `json:"status" gorm:"type:int(11);not null;default:1;index;comment:'0 disabled, 1 enabled'"`
	Sort         int    `json:"sort" gorm:"type:int(11);not null;default:0;comment:'sort order'"`
}

func (a *API) TableName() string { return "api" }
