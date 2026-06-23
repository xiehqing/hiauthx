package authorization

import (
	"context"
	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/hiauthx/db/queries"

	"github.com/xiehqing/infra/pkg/ormx"
)

type ListAuditLogsRequest struct {
	queries.AuditLogListFilter
}

func (as *Service) ListAuditLogs(ctx context.Context, req ListAuditLogsRequest) (ormx.PageResult[entity.AuditLog], error) {
	return as.queries.ListAuditLogs(ctx, req.AuditLogListFilter)
}

func (as *Service) GetAuditLog(ctx context.Context, id int64) (*entity.AuditLog, error) {
	if id <= 0 {
		return nil, ErrInvalidID
	}
	log, err := as.queries.GetAuditLog(ctx, id)
	if err != nil {
		return nil, err
	}
	if log == nil {
		return nil, ErrNotFound
	}
	return log, nil
}
