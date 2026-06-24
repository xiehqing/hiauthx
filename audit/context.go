package audit

import "context"

type contextKey struct{}

type Context struct {
	RequestID    string
	OperatorID   int64
	OperatorName string
	Method       string
	Path         string
	IP           string
	UserAgent    string
	Module       string
	Action       string
	Description  string
	ResourceType string
	Enabled      *bool
	IncludeQuery *bool
}

func WithContext(ctx context.Context, value Context) context.Context {
	return context.WithValue(ctx, contextKey{}, value)
}

func FromContext(ctx context.Context) (Context, bool) {
	value, ok := ctx.Value(contextKey{}).(Context)
	return value, ok
}

func AuditEnabledFromContext(ctx context.Context) (bool, bool) {
	value, ok := FromContext(ctx)
	if !ok || value.Enabled == nil {
		return false, false
	}
	return *value.Enabled, true
}
