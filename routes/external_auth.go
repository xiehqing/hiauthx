package routes

import (
	"context"
	"errors"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/xiehqing/hiauthx/audit"
	"github.com/xiehqing/hiauthx/authn"
	"github.com/xiehqing/hiauthx/db/entity"
	"github.com/xiehqing/infra/pkg/hertzx"
	"gorm.io/gorm"
)

const principalRequestKey = "hiauthx.authenticatedPrincipal"

// ExternalAuth is deliberately separate from the legacy constructor. All three
// components are required; partial configuration cannot fall back to local auth.
type ExternalAuth struct {
	Authenticator authn.Authenticator
	Resolver      authn.IdentityResolver
	Revoker       authn.TokenRevoker
}

func NewWithExternalAuth(db *gorm.DB, options ExternalAuth) (*Router, error) {
	if options.Authenticator == nil || options.Resolver == nil || options.Revoker == nil {
		return nil, errors.New("external authentication requires authenticator, resolver and revoker")
	}
	r := New(db)
	r.externalAuth = &options
	return r, nil
}

func (r *Router) checkExternalLogin(ctx context.Context, c *app.RequestContext) {
	identity, err := r.externalAuth.Authenticator.Authenticate(ctx, normalizeAuthorizationToken(authorizationToken(c)))
	if err != nil {
		externalError(c, err)
		return
	}
	if identity == nil || identity.Issuer == "" || identity.Subject == "" {
		externalError(c, authn.ErrUnavailable)
		return
	}
	principal, err := r.externalAuth.Resolver.Resolve(ctx, identity)
	if err != nil {
		externalError(c, err)
		return
	}
	if principal == nil || principal.LocalUserID <= 0 {
		externalError(c, authn.ErrUnavailable)
		return
	}
	user, err := r.service.GetUser(ctx, principal.LocalUserID)
	if err != nil {
		externalError(c, authn.ErrUnavailable)
		return
	}
	if user == nil || user.Status != 1 {
		externalError(c, authn.ErrForbidden)
		return
	}
	principal.Identity = *identity
	principal.Username = user.Username
	// External admin authority comes only from explicitly assigned local roles.
	principal.IsAppAdmin = explicitAppAdmin(user)
	c.Set(CtxKeyOfUser, user)
	c.Set(CtxKeyOfUserID, principal.LocalUserID)
	c.Set(CtxKeyOfUserName, principal.Username)
	c.Set(CtxKeyOfIsSystemManager, principal.IsAppAdmin)
	c.Next(bindPrincipal(ctx, c, *principal))
}

func explicitAppAdmin(user *entity.User) bool {
	if user == nil {
		return false
	}
	for _, role := range user.Roles {
		if strings.EqualFold(strings.TrimSpace(role.Name), defaultSystemRole) {
			return true
		}
	}
	return false
}

func bindPrincipal(ctx context.Context, c *app.RequestContext, principal authn.Principal) context.Context {
	c.Set(principalRequestKey, principal)
	ctx = authn.WithPrincipal(ctx, principal)
	if value, ok := audit.FromContext(ctx); ok {
		value.OperatorID = principal.LocalUserID
		value.OperatorName = principal.Username
		ctx = audit.WithContext(ctx, value)
	}
	return ctx
}

func externalError(c *app.RequestContext, err error) {
	status, message := 503, authn.ErrUnavailable.Error()
	switch {
	case errors.Is(err, authn.ErrUnauthenticated):
		status, message = 401, authn.ErrUnauthenticated.Error()
	case errors.Is(err, authn.ErrForbidden):
		status, message = 403, authn.ErrForbidden.Error()
	}
	hertzx.Abort(c, status, message)
}

// Local management behavior remains unchanged. External identities must have an
// explicit local administrator role before using administrative endpoints.
func (r *Router) checkExternalManager() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if r.externalAuth != nil {
			p, ok := authn.FromContext(ctx)
			if !ok || !p.IsAppAdmin {
				externalError(c, authn.ErrForbidden)
				return
			}
		}
		c.Next(ctx)
	}
}
