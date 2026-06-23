package entity

import "github.com/xiehqing/infra/pkg/ormx"

type Department struct {
	ormx.BaseModel
	ParentID int64  `json:"parentId" gorm:"type:bigint;not null;default:0;index;comment:'上级部门'"`
	Name     string `json:"name" gorm:"type:varchar(64);not null;comment:'部门名称'"`
	Sort     int    `json:"sort" gorm:"type:int(11);not null;default:0;comment:'排序'"`
}

func (d *Department) TableName() string {
	return "department"
}
