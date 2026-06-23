package entity

import "github.com/xiehqing/infra/pkg/ormx"

const (
	RoleCustom  = 0
	RoleBuiltIn = 1
)

type Role struct {
	ormx.BaseModel
	DisplayName string `json:"displayName" gorm:"type:varchar(64);not null;comment:'角色名称'"`
	Name        string `json:"name" gorm:"type:varchar(64);not null;uniqueIndex;comment:'角色标识'"`
	Description string `json:"description" gorm:"type:varchar(255);comment:'角色描述'"`
	BuiltIn     int    `json:"builtIn" gorm:"type:int(11);not null;default:0;index;comment:'built-in flag: 0 custom, 1 built-in'"`
	Menus       []Menu `json:"menus,omitempty" gorm:"many2many:role_menus;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (r *Role) TableName() string {
	return "role"
}
