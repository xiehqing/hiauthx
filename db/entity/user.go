package entity

import "github.com/xiehqing/infra/pkg/ormx"

type User struct {
	ormx.StatusAbleModel
	Username     string     `json:"username" gorm:"type:varchar(64);not null;uniqueIndex;comment:'用户名'"`
	Password     string     `json:"-" gorm:"type:varchar(255);not null;comment:'密码'"`
	Nickname     string     `json:"nickname" gorm:"type:varchar(64);not null;comment:'姓名'"`
	Phone        string     `json:"phone" gorm:"type:varchar(32);comment:'手机号'"`
	Email        string     `json:"email" gorm:"type:varchar(128);comment:'邮箱'"`
	WechatID     string     `json:"wechatId" gorm:"type:varchar(64);comment:'微信ID'"`
	DingTalkID   string     `json:"dingTalkId" gorm:"type:varchar(64);comment:'钉钉ID'"`
	UID          string     `json:"uid" gorm:"type:varchar(64);not null;comment:'工号'"`
	Roles        []Role     `json:"roles,omitempty" gorm:"many2many:user_roles;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	DepartmentID int64      `json:"departmentId" gorm:"type:bigint;not null;default:0;index;comment:'部门'"`
	Department   Department `json:"department,omitempty" gorm:"foreignKey:DepartmentID;references:ID;constraint:-"`
}

func (u *User) TableName() string {
	return "user"
}
