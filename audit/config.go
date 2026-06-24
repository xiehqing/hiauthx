package audit

import (
	"context"
)

func auditLogEnabled(ctx context.Context) bool {
	if value, ok := AuditEnabledFromContext(ctx); ok {
		return value
	}
	return true
}
