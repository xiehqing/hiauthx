package entity

import "github.com/xiehqing/infra/pkg/ormx"

const (
	AuditStatusSuccess = "success"
	AuditStatusFail    = "fail"

	AuditOperationCreate      = "create"
	AuditOperationUpdate      = "update"
	AuditOperationDelete      = "delete"
	AuditOperationAssociation = "association"
	AuditOperationQuery       = "query"
	AuditOperationLogin       = "login"
	AuditOperationLogout      = "logout"
)

type AuditLog struct {
	ormx.BaseModel
	RequestID    string        `json:"requestId" gorm:"type:varchar(64);index;comment:'request id'"`
	OperatorID   int64         `json:"operatorId" gorm:"type:bigint;not null;default:0;index;comment:'operator id'"`
	OperatorName string        `json:"operatorName" gorm:"type:varchar(64);index;comment:'operator name'"`
	Module       string        `json:"module" gorm:"type:varchar(64);not null;index;comment:'module'"`
	Action       string        `json:"action" gorm:"type:varchar(64);not null;index;comment:'action'"`
	ResourceType string        `json:"resourceType" gorm:"type:varchar(64);not null;index;comment:'resource type'"`
	ResourceID   int64         `json:"resourceId" gorm:"type:bigint;not null;default:0;index;comment:'resource id'"`
	Description  string        `json:"description" gorm:"type:varchar(255);comment:'description'"`
	Method       string        `json:"method" gorm:"type:varchar(16);comment:'http method'"`
	Path         string        `json:"path" gorm:"type:varchar(255);index;comment:'request path'"`
	IP           string        `json:"ip" gorm:"type:varchar(64);comment:'client ip'"`
	UserAgent    string        `json:"userAgent" gorm:"type:varchar(512);comment:'user agent'"`
	Status       string        `json:"status" gorm:"type:varchar(32);not null;index;comment:'status'"`
	ErrorMessage string        `json:"errorMessage" gorm:"type:varchar(512);comment:'error message'"`
	DurationMs   int64         `json:"durationMs" gorm:"type:bigint;not null;default:0;comment:'duration milliseconds'"`
	Changes      []AuditChange `json:"changes,omitempty" gorm:"foreignKey:AuditLogID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (a *AuditLog) TableName() string {
	return "audit_log"
}

type AuditChange struct {
	ormx.BaseModel
	AuditLogID    int64  `json:"auditLogId" gorm:"type:bigint;not null;index;comment:'audit log id'"`
	DBTableName   string `json:"tableName" gorm:"column:table_name;type:varchar(128);not null;index;comment:'table name'"`
	RecordID      int64  `json:"recordId" gorm:"type:bigint;not null;default:0;index;comment:'record id'"`
	Operation     string `json:"operation" gorm:"type:varchar(32);not null;index;comment:'operation'"`
	BeforeData    string `json:"beforeData" gorm:"type:longtext;comment:'before data json'"`
	AfterData     string `json:"afterData" gorm:"type:longtext;comment:'after data json'"`
	ChangedFields string `json:"changedFields" gorm:"type:text;comment:'changed fields json'"`
	FieldChanges  string `json:"fieldChanges" gorm:"type:longtext;comment:'field changes json with before and after values'"`
}

func (a *AuditChange) TableName() string {
	return "audit_change"
}
