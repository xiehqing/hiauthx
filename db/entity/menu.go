package entity

import "github.com/xiehqing/infra/pkg/ormx"

const (
	MenuTypeGroup  = 0
	MenuTypeMenu   = 1
	MenuTypeButton = 2

	MenuHidden = 0
	MenuShown  = 1
)

type Menu struct {
	ormx.BaseModel
	Type     int    `json:"type" gorm:"type:int(11);not null;comment:'类型：菜单1，按钮2'"`
	ParentID int64  `json:"parentId" gorm:"type:bigint;not null;default:0;index;comment:'上级菜单'"`
	Name     string `json:"name" gorm:"type:varchar(64);not null;comment:'菜单名称'"`
	Route    string `json:"route" gorm:"type:varchar(255);comment:'路由'"`
	Sort     int    `json:"sort" gorm:"type:int(11);not null;default:0;comment:'排序'"`
	Icon     string `json:"icon" gorm:"type:varchar(128);comment:'图标'"`
	Show     int    `json:"show" gorm:"type:int(11);not null;comment:'是否显示：0不显示，1显示'"`
}

func (m *Menu) TableName() string {
	return "menu"
}
