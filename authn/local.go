package authn

import (
	"context"
	"strings"

	"github.com/xiehqing/hitoken/htputil"
)

// LocalAuthenticator delegates to the existing process-local token manager.
type LocalAuthenticator struct{}

func (LocalAuthenticator) Authenticate(_ context.Context, token string) (*Identity, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrUnauthenticated
	}
	if err := htputil.CheckLogin(token); err != nil {
		return nil, ErrUnauthenticated
	}
	subject, err := htputil.GetLoginID(token)
	if err != nil || strings.TrimSpace(subject) == "" {
		return nil, ErrUnauthenticated
	}
	return &Identity{Issuer: "local", Subject: subject}, nil
}

func (LocalAuthenticator) Revoke(_ context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return ErrUnauthenticated
	}
	return htputil.LogoutByToken(token)
}
