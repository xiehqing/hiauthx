package routes

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/xiehqing/hiauthx/audit"
	"github.com/xiehqing/hiauthx/authn"
	"github.com/xiehqing/hiauthx/db/entity"
)

func TestExternalAdminNameDoesNotGrantLocalAuthority(t *testing.T) {
	if explicitAppAdmin(&entity.User{Username: "admin"}) {
		t.Fatal("external username granted admin")
	}
	if !explicitAppAdmin(&entity.User{Username: "ext_123", Roles: []entity.Role{{Name: "role_admin"}}}) {
		t.Fatal("explicit local role was ignored")
	}
}

type failingAuth struct{ err error }

func (f failingAuth) Authenticate(context.Context, string) (*authn.Identity, error) {
	return nil, f.err
}
func (f failingAuth) Revoke(context.Context, string) error { return f.err }

func TestExternalAuthenticationDoesNotFallBack(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
	}{
		{authn.ErrUnauthenticated, 401}, {authn.ErrForbidden, 403}, {authn.ErrUnavailable, 503}, {errors.New("private upstream body"), 503},
	} {
		r := &Router{externalAuth: &ExternalAuth{Authenticator: failingAuth{tc.err}}}
		s := server.New()
		reached := false
		s.GET("/test", r.CheckLogin(), func(context.Context, *app.RequestContext) { reached = true })
		response := ut.PerformRequest(s.Engine, "GET", "/test", nil, ut.Header{Key: "Authorization", Value: "Bearer token"})
		if response.Code != tc.status || reached {
			t.Fatalf("status=%d reached=%v", response.Code, reached)
		}
	}
}

func TestPrincipalBindsBothBusinessAndRequestAudit(t *testing.T) {
	c := app.NewContext(0)
	ctx := audit.WithContext(context.Background(), audit.Context{RequestID: "req-1"})
	p := authn.Principal{LocalUserID: 7, Username: "external-user"}
	ctx = bindPrincipal(ctx, c, p)
	current, ok := authn.FromContext(ctx)
	if !ok || current.LocalUserID != 7 {
		t.Fatal("missing principal")
	}
	a, ok := audit.FromContext(ctx)
	if !ok || a.OperatorID != 7 || a.OperatorName != "external-user" || a.RequestID != "req-1" {
		t.Fatal("business audit lost identity or request metadata")
	}
	if id, ok := currentUserID(c); !ok || id != 7 {
		t.Fatal("currentUserID tried local token parsing")
	}
}

func TestExternalRoutesDoNotExposeLocalLoginOrIdentityMutation(t *testing.T) {
	r := &Router{externalAuth: &ExternalAuth{}}
	s := server.New()
	api := s.Group("/api/v1")
	r.registerAuthenticationRoutes(api)
	r.registerUserRoutes(api)
	r.registerDepartmentRoutes(api)
	for _, path := range []string{"/auth/login", "/auth/encrypt-config", "/users", "/departments"} {
		method := "POST"
		if path == "/auth/encrypt-config" {
			method = "GET"
		}
		resp := ut.PerformRequest(s.Engine, method, "/api/v1"+path, nil)
		if resp.Code != 404 && resp.Code != 405 {
			t.Fatalf("%s is exposed: %d", path, resp.Code)
		}
	}
}

func TestPartialExternalConfigurationRejected(t *testing.T) {
	if _, err := NewWithExternalAuth(nil, ExternalAuth{}); err == nil {
		t.Fatal("partial configuration accepted")
	}
}
