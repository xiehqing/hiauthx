package queries

import (
	"context"
	"fmt"
	"github.com/xiehqing/hiauthx/audit"
	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/infra/pkg/ormx"
	"gorm.io/gorm"
	"time"
)

// AuditLogListFilter defines query conditions for paged audit log lookup.
type AuditLogListFilter struct {
	ormx.Pagination
	OperatorName string `json:"operatorName" form:"operatorName"`
	Module       string `json:"module" form:"module"`
	Action       string `json:"action" form:"action"`
	ResourceType string `json:"resourceType" form:"resourceType"`
	ResourceID   int64  `json:"resourceId" form:"resourceId"`
	Status       string `json:"status" form:"status"`
	StartTime    *time.Time
	EndTime      *time.Time
}

// OperationLog is a lightweight view of audit_log for user operation behavior display.
type OperationLog struct {
	ID           int64     `json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	OperatorID   int64     `json:"operatorId"`
	OperatorName string    `json:"operatorName"`
	Module       string    `json:"module"`
	Action       string    `json:"action"`
	Description  string    `json:"description"`
	Method       string    `json:"method"`
	Path         string    `json:"path"`
	IP           string    `json:"ip"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"errorMessage"`
	DurationMs   int64     `json:"durationMs"`
}

// RequestAudit describes an operation-level audit that is not produced by a GORM data callback.
type RequestAudit struct {
	Module       string
	Action       string
	ResourceType string
	ResourceID   int64
	Description  string
	Status       string
	ErrorMessage string
	DurationMs   int64
}

func (q *Queries) AuditAssociation(ctx context.Context, tx *gorm.DB, resourceType, tableName string, recordID int64, before, after any) error {
	return q.audit.AuditAssociation(ctx, tx, resourceType, tableName, recordID, before, after)
}

// CreateRequestAudit persists a request-level audit entry such as login, logout, or query.
func (q *Queries) CreateRequestAudit(ctx context.Context, request RequestAudit) error {
	if !q.GetSystemConfigBoolValueByKey(ctx, entity.AuditLogEnabled, true) {
		return nil
	}
	if requestAuditAction(request.Action) == entity.AuditOperationQuery && !q.GetSystemConfigBoolValueByKey(ctx, entity.AuditLogIncludeQuery, true) {
		return nil
	}
	auditContext, _ := audit.FromContext(ctx)
	log := entity.AuditLog{
		RequestID:    auditContext.RequestID,
		OperatorID:   auditContext.OperatorID,
		OperatorName: auditContext.OperatorName,
		Module:       request.Module,
		Action:       request.Action,
		ResourceType: request.ResourceType,
		ResourceID:   request.ResourceID,
		Description:  request.Description,
		Method:       auditContext.Method,
		Path:         auditContext.Path,
		IP:           auditContext.IP,
		UserAgent:    auditContext.UserAgent,
		Status:       request.Status,
		ErrorMessage: request.ErrorMessage,
		DurationMs:   request.DurationMs,
	}
	if log.Module == "" {
		log.Module = audit.GetAuditModule(request.ResourceType)
	}
	if log.Action == "" {
		log.Action = entity.AuditOperationQuery
	}
	if log.ResourceType == "" {
		log.ResourceType = "request"
	}
	if log.Status == "" {
		log.Status = entity.AuditStatusSuccess
	}
	if log.Description == "" {
		log.Description = auditRequestDescription(log.Action, log.ResourceType, log.Path)
	}
	return q.db.Session(&gorm.Session{SkipHooks: true}).WithContext(ctx).Create(&log).Error
}

func requestAuditAction(action string) string {
	if action == "" {
		return entity.AuditOperationQuery
	}
	return action
}

// auditRequestDescription builds the default description for request-level audit logs.
func auditRequestDescription(action, resourceType, path string) string {
	switch action {
	case entity.AuditOperationLogin:
		return "用户登录"
	case entity.AuditOperationLogout:
		return "退出登录"
	case entity.AuditOperationQuery:
		return fmt.Sprintf("查询%s", audit.GetAuditResourceName(resourceType))
	default:
		return fmt.Sprintf("%s接口：%s", action, path)
	}
}

// GetAuditLog returns one audit log and preloads its database change details.
func (q *Queries) GetAuditLog(ctx context.Context, id int64) (*entity.AuditLog, error) {
	var log entity.AuditLog
	err := q.db.WithContext(ctx).Preload("Changes").First(&log, "id = ?", id).Error
	if err != nil {
		return nil, ormx.NotFoundAsNil(err)
	}
	return &log, nil
}

// ListAuditLogs returns paged audit logs using keyword, resource, status, and time filters.
func (q *Queries) ListAuditLogs(ctx context.Context, filter AuditLogListFilter) (ormx.PageResult[entity.AuditLog], error) {
	db := q.db.WithContext(ctx).Model(&entity.AuditLog{})
	db = applyAuditLogFilters(db, filter)

	return ormx.Paginate[entity.AuditLog](db, filter.Pagination, auditLogSortMap())
}

func (q *Queries) ListOperationLogs(ctx context.Context, filter AuditLogListFilter) (ormx.PageResult[OperationLog], error) {
	db := q.db.WithContext(ctx).
		Model(&entity.AuditLog{}).
		Select("id", "created_at", "operator_id", "operator_name", "module", "action", "description", "method", "path", "ip", "status", "error_message", "duration_ms")
	db = applyAuditLogFilters(db, filter)

	return ormx.Paginate[OperationLog](db, filter.Pagination, auditLogSortMap())
}

func applyAuditLogFilters(db *gorm.DB, filter AuditLogListFilter) *gorm.DB {
	if ormx.KeywordPresent(filter.Keyword) {
		keyword := ormx.LikeKeyword(filter.Keyword)
		db = db.Where("operator_name LIKE ? OR description LIKE ? OR path LIKE ?", keyword, keyword, keyword)
	}
	if filter.OperatorName != "" {
		db = db.Where("operator_name LIKE ?", ormx.LikeKeyword(filter.OperatorName))
	}
	if filter.Module != "" {
		db = db.Where("module = ?", filter.Module)
	}
	if filter.Action != "" {
		db = db.Where("action = ?", filter.Action)
	}
	if filter.ResourceType != "" {
		db = db.Where("resource_type = ?", filter.ResourceType)
	}
	if filter.ResourceID > 0 {
		db = db.Where("resource_id = ?", filter.ResourceID)
	}
	if filter.Status != "" {
		db = db.Where("status = ?", filter.Status)
	}
	if filter.StartTime != nil {
		db = db.Where("created_at >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		db = db.Where("created_at <= ?", *filter.EndTime)
	}
	return db
}

func auditLogSortMap() map[string]string {
	return map[string]string{
		"id":           "id",
		"operatorName": "operator_name",
		"module":       "module",
		"action":       "action",
		"resourceType": "resource_type",
		"resourceId":   "resource_id",
		"status":       "status",
		"durationMs":   "duration_ms",
		"createdAt":    "created_at",
		"updatedAt":    "updated_at",
	}
}
