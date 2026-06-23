package audit

import (
	"github.com/xiehqing/hiauthx/db/entity"
	"gorm.io/gorm"
)

// RegisterAuditHooks 注册审计钩子
func (q *Queries) RegisterAuditHooks() {
	if q == nil || q.db == nil {
		return
	}

	create := q.db.Callback().Create()
	if create.Get(auditCreateCallbackName) == nil {
		_ = create.After("gorm:create").Register(auditCreateCallbackName, q.afterCreate)
	}

	update := q.db.Callback().Update()
	if update.Get(auditUpdateBeforeCallbackName) == nil {
		_ = update.Before("gorm:update").Register(auditUpdateBeforeCallbackName, q.beforeUpdate)
	}
	if update.Get(auditUpdateAfterCallbackName) == nil {
		_ = update.After("gorm:update").Register(auditUpdateAfterCallbackName, q.afterUpdate)
	}

	deleteCallback := q.db.Callback().Delete()
	if deleteCallback.Get(auditDeleteBeforeCallbackName) == nil {
		_ = deleteCallback.Before("gorm:delete").Register(auditDeleteBeforeCallbackName, q.beforeDelete)
	}
	if deleteCallback.Get(auditDeleteAfterCallbackName) == nil {
		_ = deleteCallback.After("gorm:delete").Register(auditDeleteAfterCallbackName, q.afterDelete)
	}
}

// afterCreate 对象创建审计
func (q *Queries) afterCreate(tx *gorm.DB) {
	if !shouldAudit(tx) {
		return
	}
	cfg, ok := auditTableConfigOf(tx)
	if !ok {
		return
	}
	recordIds := getAuditDataPrimaryIDs(tx)
	if len(recordIds) == 0 {
		return
	}
	if len(recordIds) > maxAuditBatchSize {
		recordIds = recordIds[:maxAuditBatchSize]
	}
	afterItems, err := buildAuditSnapshots(tx, cfg, recordIds)
	if err != nil {
		tx.AddError(err)
		return
	}
	for _, recordId := range recordIds {
		after := afterItems[recordId]
		if after == nil {
			continue
		}
		if err := q.RecordAudit(tx.Statement.Context, tx, auditRecordOptions{
			module:       GetAuditModule(cfg.resourceType),
			action:       entity.AuditOperationCreate,
			resourceType: cfg.resourceType,
			resourceID:   recordId,
			tableName:    tx.Statement.Table,
			operation:    entity.AuditOperationCreate,
			description:  auditDescription(cfg.resourceType, entity.AuditOperationCreate, recordId),
			after:        after,
		}); err != nil {
			tx.AddError(err)
			return
		}
	}
}

func (q *Queries) beforeUpdate(tx *gorm.DB) {
	if !shouldAudit(tx) {
		return
	}
	cfg, ok := auditTableConfigOf(tx)
	if !ok {
		return
	}
	snapshots, err := buildBeforeAuditSnapshot(tx, cfg)
	if err != nil {
		tx.AddError(err)
		return
	}
	if len(snapshots.IDs) == 0 {
		return
	}
	tx.InstanceSet(auditBeforeKey, snapshots)
}

func (q *Queries) afterUpdate(tx *gorm.DB) {
	if !shouldAudit(tx) {
		return
	}
	cfg, ok := auditTableConfigOf(tx)
	if !ok {
		return
	}
	beforeSet, ok := auditBeforeSnapshot(tx)
	if !ok {
		return
	}
	if tx.RowsAffected == 0 {
		return
	}
	afterItems, err := buildAuditSnapshots(tx, cfg, beforeSet.IDs)
	if err != nil {
		tx.AddError(err)
		return
	}
	for _, recordID := range beforeSet.IDs {
		before := beforeSet.Items[recordID]
		after := afterItems[recordID]
		if before == nil && after == nil {
			continue
		}
		if err := q.RecordAudit(tx.Statement.Context, tx, auditRecordOptions{
			module:       GetAuditModule(cfg.resourceType),
			action:       entity.AuditOperationUpdate,
			resourceType: cfg.resourceType,
			resourceID:   recordID,
			tableName:    tx.Statement.Table,
			operation:    entity.AuditOperationUpdate,
			description:  auditDescription(cfg.resourceType, entity.AuditOperationUpdate, recordID),
			before:       before,
			after:        after,
		}); err != nil {
			tx.AddError(err)
			return
		}
	}
}

// beforeDelete captures rows before GORM deletes them so the old values can be logged.
func (q *Queries) beforeDelete(tx *gorm.DB) {
	if !shouldAudit(tx) {
		return
	}
	cfg, ok := auditTableConfigOf(tx)
	if !ok {
		return
	}
	snapshots, err := buildBeforeAuditSnapshot(tx, cfg)
	if err != nil {
		tx.AddError(err)
		return
	}
	if len(snapshots.IDs) == 0 {
		return
	}
	tx.InstanceSet(auditBeforeKey, snapshots)
}

// afterDelete writes one audit log per deleted record id using the captured old values.
func (q *Queries) afterDelete(tx *gorm.DB) {
	if !shouldAudit(tx) {
		return
	}
	cfg, ok := auditTableConfigOf(tx)
	if !ok {
		return
	}
	beforeSet, ok := auditBeforeSnapshot(tx)
	if !ok {
		return
	}
	if tx.RowsAffected == 0 {
		return
	}
	for _, recordID := range beforeSet.IDs {
		before := beforeSet.Items[recordID]
		if before == nil {
			continue
		}
		if err := q.RecordAudit(tx.Statement.Context, tx, auditRecordOptions{
			module:       GetAuditModule(cfg.resourceType),
			action:       entity.AuditOperationDelete,
			resourceType: cfg.resourceType,
			resourceID:   recordID,
			tableName:    tx.Statement.Table,
			operation:    entity.AuditOperationDelete,
			description:  auditDescription(cfg.resourceType, entity.AuditOperationDelete, recordID),
			before:       before,
		}); err != nil {
			tx.AddError(err)
			return
		}
	}
}

// auditBeforeSnapshot reads the before-image snapshot stored on the GORM statement.
func auditBeforeSnapshot(tx *gorm.DB) (auditSnapshotSet, bool) {
	value, ok := tx.InstanceGet(auditBeforeKey)
	if !ok {
		return auditSnapshotSet{}, false
	}
	snapshots, ok := value.(auditSnapshotSet)
	return snapshots, ok && len(snapshots.IDs) > 0 && len(snapshots.Items) > 0
}
