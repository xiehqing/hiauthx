package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/xiehqing/hiauthx/db/entity"
	"gorm.io/gorm"
	"reflect"
	"sort"
	"strings"
)

type Queries struct {
	db *gorm.DB
}

func NewAudit(db *gorm.DB) *Queries {
	q := &Queries{
		db: db,
	}
	q.RegisterAuditHooks()
	return q
}

// RecordAudit 记录日志
func (q *Queries) RecordAudit(ctx context.Context, tx *gorm.DB, options auditRecordOptions) error {
	auditTx := cleanAuditDB(tx, ctx)
	if !auditLogEnabled(ctx) {
		return nil
	}
	auditContext, _ := FromContext(ctx)
	beforeData, beforeMap := serializeAuditDataWithMask(options.before, true)
	afterData, afterMap := serializeAuditDataWithMask(options.after, true)
	_, beforeRawMap := serializeAuditDataWithMask(options.before, false)
	_, afterRawMap := serializeAuditDataWithMask(options.after, false)
	changedFields := serializeChangedFields(beforeRawMap, afterRawMap)
	fieldChanges := serializeChangedFieldsWithValues(beforeRawMap, afterRawMap, beforeMap, afterMap)
	log := entity.AuditLog{
		RequestID:    auditContext.RequestID,
		OperatorID:   auditContext.OperatorID,
		OperatorName: auditContext.OperatorName,
		Module:       options.module,
		Action:       options.action,
		ResourceType: options.resourceType,
		ResourceID:   options.resourceID,
		Description:  options.description,
		Method:       auditContext.Method,
		Path:         auditContext.Path,
		IP:           auditContext.IP,
		UserAgent:    auditContext.UserAgent,
		Status:       entity.AuditStatusSuccess,
	}
	if auditContext.Module != "" {
		log.Module = auditContext.Module
	}
	if auditContext.Action != "" {
		log.Action = auditContext.Action
	}
	if auditContext.Description != "" {
		log.Description = auditContext.Description
	}
	if auditContext.ResourceType != "" {
		log.ResourceType = auditContext.ResourceType
	}
	if log.Module == "" {
		log.Module = "系统"
	}
	if log.Description == "" {
		log.Description = auditDescription(options.resourceType, options.action, options.resourceID)
	}
	change := entity.AuditChange{
		DBTableName:   options.tableName,
		RecordID:      options.resourceID,
		Operation:     options.operation,
		BeforeData:    beforeData,
		AfterData:     afterData,
		ChangedFields: changedFields,
		FieldChanges:  fieldChanges,
	}
	if err := auditTx.Create(&log).Error; err != nil {
		return err
	}
	change.AuditLogID = log.ID
	return auditTx.Create(&change).Error
}

// AuditAssociation writes audit data for explicit authorization relationship changes.
func (q *Queries) AuditAssociation(ctx context.Context, tx *gorm.DB, resourceType, tableName string, recordID int64, before, after any) error {
	return q.RecordAudit(ctx, tx, auditRecordOptions{
		module:       GetAuditModule(resourceType),
		action:       entity.AuditOperationAssociation,
		resourceType: resourceType,
		resourceID:   recordID,
		tableName:    tableName,
		operation:    entity.AuditOperationUpdate,
		description:  auditDescription(resourceType, entity.AuditOperationAssociation, recordID),
		before:       before,
		after:        after,
	})
}

// auditDescription builds the default human-readable description for a data-change audit.
func auditDescription(resourceType, action string, resourceID int64) string {
	actionName := map[string]string{
		entity.AuditOperationCreate:      "新增",
		entity.AuditOperationUpdate:      "修改",
		entity.AuditOperationDelete:      "删除",
		entity.AuditOperationAssociation: "关联",
	}[action]
	if actionName == "" {
		actionName = action
	}
	return fmt.Sprintf("%s%s，ID：%d", actionName, GetAuditResourceName(resourceType), resourceID)
}

// GetAuditResourceName maps a resource type to its display name.
func GetAuditResourceName(resourceType string) string {
	if cfg, ok := auditTables[resourceType]; ok {
		return cfg.resourceName
	}
	return resourceType
}

// serializeAuditDataWithMask serializes audit data with masking.
func serializeAuditDataWithMask(value any, mask bool) (string, map[string]any) {
	if value == nil {
		return "", nil
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", nil
	}

	var data map[string]any
	if err := json.Unmarshal(bytes, &data); err != nil {
		return string(bytes), nil
	}
	data = removeAuditAssociationFields(data)
	if mask {
		sensitiveAuditDataMap(data)
	}
	bytes, err = json.Marshal(data)
	if err != nil {
		return "", data
	}
	return string(bytes), data
}

// removeAuditAssociationFields removes association fields that are not part of the audited business row.
func removeAuditAssociationFields(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}
	delete(data, "roles")
	delete(data, "menus")
	delete(data, "department")
	return data
}

// sensitiveAuditDataMap recursively masks sensitive values in audit payloads.
func sensitiveAuditDataMap(data map[string]any) {
	configKey := ""
	if value, ok := data["key"].(string); ok {
		configKey = strings.ToLower(value)
	}
	for key, value := range data {
		lowerKey := strings.ToLower(key)
		if sensitiveAuditField(lowerKey) || (key == "value" && sensitiveConfigKey(configKey)) {
			data[key] = "******"
			continue
		}
		switch item := value.(type) {
		case map[string]any:
			sensitiveAuditDataMap(item)
		case []any:
			for _, child := range item {
				if childMap, ok := child.(map[string]any); ok {
					sensitiveAuditDataMap(childMap)
				}
			}
		}
	}
}

// sensitiveAuditField reports whether a field name should be masked in audit data.
func sensitiveAuditField(key string) bool {
	return strings.Contains(key, "password") ||
		strings.Contains(key, "secret") ||
		strings.Contains(key, "token") ||
		strings.Contains(key, "privatekey") ||
		strings.Contains(key, "private_key") ||
		strings.Contains(key, "authorization")
}

// sensitiveConfigKey reports whether a system config key stores sensitive content.
func sensitiveConfigKey(key string) bool {
	return strings.Contains(key, "password") ||
		strings.Contains(key, "secret") ||
		strings.Contains(key, "token") ||
		strings.Contains(key, "private_key") ||
		strings.Contains(key, "privatekey") ||
		strings.Contains(key, "storage")
}

// serializeChangedFields returns a JSON array of changed business field names.
func serializeChangedFields(before, after map[string]any) string {
	changes := getFieldChangesWithValues(before, after, before, after)
	fields := make([]string, 0, len(changes))
	for _, change := range changes {
		fields = append(fields, change.Field)
	}
	bytes, err := json.Marshal(fields)
	if err != nil {
		return "[]"
	}
	return string(bytes)
}

// serializeChangedFieldsWithValues compares raw values while returning masked display values.
func serializeChangedFieldsWithValues(before, after, beforeValues, afterValues map[string]any) string {
	changes := getFieldChangesWithValues(before, after, beforeValues, afterValues)
	bytes, err := json.Marshal(changes)
	if err != nil {
		return "[]"
	}
	return string(bytes)
}

// ignoreAuditCompareField reports whether a metadata field should be excluded from comparisons.
func ignoreAuditCompareField(field string) bool {
	switch field {
	case "createdAt", "createdBy", "updatedAt", "updatedBy":
		return true
	default:
		return false
	}
}

// valueOfAuditField safely reads a field from an audit data map.
func valueOfAuditField(data map[string]any, field string) any {
	if data == nil {
		return nil
	}
	return data[field]
}

// getFieldChangesWithValues compares raw maps and reads output values from the provided display maps.
func getFieldChangesWithValues(before, after, beforeValues, afterValues map[string]any) []auditFieldChange {
	if before == nil && after == nil {
		return []auditFieldChange{}
	}

	keys := make(map[string]struct{}, len(before)+len(after))
	for key := range before {
		if !ignoreAuditCompareField(key) {
			keys[key] = struct{}{}
		}
	}
	for key := range after {
		if !ignoreAuditCompareField(key) {
			keys[key] = struct{}{}
		}
	}

	fields := make([]string, 0, len(keys))
	for field := range keys {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	result := make([]auditFieldChange, 0, len(fields))
	for _, field := range fields {
		beforeValue := valueOfAuditField(before, field)
		afterValue := valueOfAuditField(after, field)
		if !reflect.DeepEqual(beforeValue, afterValue) {
			result = append(result, auditFieldChange{
				Field:  field,
				Before: valueOfAuditField(beforeValues, field),
				After:  valueOfAuditField(afterValues, field),
			})
		}
	}
	return result
}
