package audit

import (
	"context"
	"github.com/xiehqing/hiauthx/db/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"reflect"
	"strings"
)

const (
	auditBeforeKey = "hi_auth:audit:before"

	auditCreateCallbackName       = "hi_auth:audit:after_create"
	auditUpdateBeforeCallbackName = "hi_auth:audit:before_update"
	auditUpdateAfterCallbackName  = "hi_auth:audit:after_update"
	auditDeleteBeforeCallbackName = "hi_auth:audit:before_delete"
	auditDeleteAfterCallbackName  = "hi_auth:audit:after_delete"

	maxAuditBatchSize = 500
)

type auditTableConfig struct {
	resourceType string
	resourceName string
	module       string
	model        any
}

var auditTables = make(map[string]auditTableConfig)

func auditTableConfigOf(tx *gorm.DB) (auditTableConfig, bool) {
	cfg, ok := auditTables[tx.Statement.Table]
	return cfg, ok
}

func RegisterAuditTable(resourceType, module, resourceName string, model any) {
	auditTables[resourceType] = auditTableConfig{resourceType, module, resourceName, model}
}

func init() {
	RegisterAuditTable("department", "部门管理", "部门", &entity.Department{})
	RegisterAuditTable("role", "角色管理", "角色", &entity.Role{})
	RegisterAuditTable("user", "用户管理", "用户", &entity.User{})
	RegisterAuditTable("menu", "菜单管理", "菜单", &entity.Menu{})
	RegisterAuditTable("system_config", "系统配置管理", "配置", &entity.SystemConfig{})
	RegisterAuditTable("api", "API 管理", "API", &entity.API{})
}

type auditRecordOptions struct {
	module       string
	action       string
	resourceType string
	resourceID   int64
	tableName    string
	operation    string
	description  string
	before       any
	after        any
}

// auditFieldChange describes one changed business field with before and after values.
type auditFieldChange struct {
	Field  string `json:"field"`
	Before any    `json:"before"`
	After  any    `json:"after"`
}

func cleanAuditDB(tx *gorm.DB, ctx context.Context) *gorm.DB {
	db := tx.Session(&gorm.Session{NewDB: true, SkipHooks: true})
	db.Statement = &gorm.Statement{
		DB:        db,
		ConnPool:  tx.Statement.ConnPool,
		Context:   ctx,
		Clauses:   map[string]clause.Clause{},
		Preloads:  map[string][]interface{}{},
		SkipHooks: true,
		Unscoped:  tx.Statement.Unscoped,
		Settings:  tx.Statement.Settings,
	}
	return db
}

func shouldAudit(tx *gorm.DB) bool {
	if tx == nil || tx.Statement == nil || tx.Statement.Schema == nil || tx.Statement.Table == "" {
		return false
	}
	if _, ok := auditTables[tx.Statement.Table]; !ok {
		return false
	}
	if strings.HasPrefix(tx.Statement.Table, "audit_") {
		return false
	}
	if !auditLogEnabled(tx.Statement.Context) {
		return false
	}
	return true
}

// getAuditDataPrimaryIDsForChange 获取审计数据的主键值列表
func getAuditDataPrimaryIDsForChange(tx *gorm.DB, cfg auditTableConfig) ([]int64, error) {
	recordIDs := getAuditDataPrimaryIDs(tx)
	if len(recordIDs) > 0 {
		return recordIDs, nil
	}
	return getAuditDataPrimaryIDsFromWhere(tx, cfg)
}

// getAuditDataPrimaryIDsFromWhere 从 where 子句中获取审计数据的主键值列表
func getAuditDataPrimaryIDsFromWhere(tx *gorm.DB, cfg auditTableConfig) ([]int64, error) {
	where, ok := tx.Statement.Clauses["WHERE"]
	if !ok {
		return nil, nil
	}
	var recordIDs []int64
	db := cleanAuditDB(tx, context.WithoutCancel(tx.Statement.Context)).Model(cfg.model)
	db.Statement.Clauses["WHERE"] = where
	err := db.Pluck("id", &recordIDs).Error
	if err != nil {
		return nil, err
	}
	return uniqueAuditDataIDs(recordIDs), nil
}

// getAuditDataPrimaryIDs 获取审计数据的主键值列表
func getAuditDataPrimaryIDs(tx *gorm.DB) []int64 {
	if tx.Statement == nil || tx.Statement.Schema == nil {
		return nil
	}
	field := tx.Statement.Schema.PrioritizedPrimaryField
	if field == nil {
		return nil
	}
	if ids := getAuditPrimaryIDsFromValue(tx.Statement.ReflectValue, field); len(ids) > 0 {
		return ids
	}
	if ids := getAuditPrimaryIDsFromValue(reflect.ValueOf(tx.Statement.Dest), field); len(ids) > 0 {
		return ids
	}
	if ids := getAuditPrimaryIDsFromValue(reflect.ValueOf(tx.Statement.Model), field); len(ids) > 0 {
		return ids
	}
	return nil
}

// getAuditPrimaryIDFromValue 从 value 中获取审计主键值列表
func getAuditPrimaryIDsFromValue(value reflect.Value, field *schema.Field) []int64 {
	if !value.IsValid() {
		return nil
	}
	// 如果是指针或 interface，就不断解引用：
	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		result := make([]int64, 0, value.Len())
		for i := 0; i < value.Len(); i++ {
			result = append(result, getAuditPrimaryIDsFromValue(value.Index(i), field)...)
		}
		return uniqueAuditDataIDs(result)
	}
	if value.Kind() != reflect.Struct {
		return nil
	}
	fieldValue := value.FieldByName(field.Name)
	if !fieldValue.IsValid() {
		return nil
	}
	for fieldValue.Kind() == reflect.Pointer || fieldValue.Kind() == reflect.Interface {
		if fieldValue.IsNil() {
			return nil
		}
		fieldValue = fieldValue.Elem()
	}
	switch fieldValue.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		id := fieldValue.Int()
		if id > 0 {
			return []int64{id}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		id := int64(fieldValue.Uint())
		if id > 0 {
			return []int64{id}
		}
	}
	return nil
}

func uniqueAuditDataIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func newAuditModelSlice(cfg auditTableConfig) any {
	t := reflect.TypeOf(cfg.model)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return reflect.New(reflect.SliceOf(t)).Interface()
}

// auditSnapshotSet stores the record ids and snapshots captured before a write.
type auditSnapshotSet struct {
	IDs    []int64
	Items  map[int64]any
	TooBig bool
}

// buildBeforeAuditSnapshot 构建审计快照
func buildBeforeAuditSnapshot(tx *gorm.DB, cfg auditTableConfig) (auditSnapshotSet, error) {
	recordIds, err := getAuditDataPrimaryIDsForChange(tx, cfg)
	if err != nil {
		return auditSnapshotSet{}, err
	}
	if len(recordIds) > maxAuditBatchSize {
		return auditSnapshotSet{TooBig: true}, nil
	}
	items, err := buildAuditSnapshots(tx, cfg, recordIds)
	if err != nil {
		return auditSnapshotSet{}, err
	}
	return auditSnapshotSet{IDs: recordIds, Items: items}, nil
}

// buildAuditSnapshots 构建审计快照
func buildAuditSnapshots(tx *gorm.DB, cfg auditTableConfig, recordIDs []int64) (map[int64]any, error) {
	recordIDs = uniqueAuditDataIDs(recordIDs)
	if len(recordIDs) == 0 {
		return map[int64]any{}, nil
	}
	slice := newAuditModelSlice(cfg)
	err := cleanAuditDB(tx, context.WithoutCancel(tx.Statement.Context)).
		Model(cfg.model).
		Where("id IN ?", recordIDs).
		Find(slice).
		Error
	if err != nil {
		return nil, err
	}
	return buildAuditSnapshotMap(slice, tx.Statement.Schema.PrioritizedPrimaryField)
}

func buildAuditSnapshotMap(slice any, field *schema.Field) (map[int64]any, error) {
	result := make(map[int64]any)
	value := reflect.ValueOf(slice)
	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return result, nil
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Slice && value.Kind() != reflect.Array {
		return result, nil
	}
	for i := 0; i < value.Len(); i++ {
		item := value.Index(i)
		ids := getAuditPrimaryIDsFromValue(item, field)
		if len(ids) == 0 {
			continue
		}
		itemValue := item
		if itemValue.CanAddr() {
			result[ids[0]] = itemValue.Addr().Interface()
		} else {
			result[ids[0]] = itemValue.Interface()
		}
	}
	return result, nil
}

// GetAuditModule maps a resource type to the module name stored in audit logs.
func GetAuditModule(resourceType string) string {
	if cfg, ok := auditTables[resourceType]; ok {
		return cfg.resourceName
	}
	return "系统"
}
