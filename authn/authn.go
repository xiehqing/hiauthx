// Package authn defines the authentication boundary shared by local and external providers.
package authn

import (
	"context"
	"errors"
	"time"
)

var (
	ErrUnauthenticated = errors.New("登录状态已失效，请重新登录")
	ErrForbidden       = errors.New("无权访问当前应用")
	ErrUnavailable     = errors.New("认证服务暂不可用")
)

type Identity struct {
	Issuer      string
	Subject     string
	Username    string
	DisplayName string
	ExpiresAt   *time.Time
}

type Principal struct {
	LocalUserID int64
	Identity    Identity
	Username    string
	IsAppAdmin  bool
}

// Authenticate must verify both credentials and application admission.
// Implementations must never return raw credentials in Identity or errors.
type Authenticator interface {
	Authenticate(context.Context, string) (*Identity, error)
}

type IdentityResolver interface {
	Resolve(context.Context, *Identity) (*Principal, error)
}

type TokenRevoker interface {
	Revoke(context.Context, string) error
}

type principalKey struct{}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}
