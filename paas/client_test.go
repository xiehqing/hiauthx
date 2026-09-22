package paas

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xiehqing/hiauthx/authn"
)

const validReply = `{"code":0,"data":{"active":true,"allowed":true,"appCode":"agent","issuer":"paas","subject":"user_1","user":{"username":"admin","displayName":"外部用户"}}}`

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	s := httptest.NewTLSServer(handler)
	t.Cleanup(s.Close)
	c, err := newClient(Config{BaseURL: s.URL, IssuerID: "paas", AppCode: "agent"}, "app-secret", s.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestEveryRequestRechecksTokenAndSendsAppCredentials(t *testing.T) {
	var calls atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/open/auth/check" || r.Method != "POST" {
			t.Error("unexpected endpoint")
		}
		user, secret, ok := r.BasicAuth()
		if !ok || user != "agent" || secret != "app-secret" {
			t.Error("missing application credentials")
		}
		var body map[string]string
		if json.NewDecoder(r.Body).Decode(&body) != nil || body["accessToken"] != "user-token" {
			t.Error("missing user token")
		}
		if calls.Add(1) == 1 {
			_, _ = w.Write([]byte(validReply))
		} else {
			_, _ = w.Write([]byte(`{"code":0,"data":{"active":false}}`))
		}
	})
	identity, err := c.Authenticate(context.Background(), "user-token")
	if err != nil || identity.Subject != "user_1" {
		t.Fatalf("identity=%v err=%v", identity, err)
	}
	if _, err := c.Authenticate(context.Background(), "user-token"); !errors.Is(err, authn.ErrUnauthenticated) {
		t.Fatalf("revocation ignored: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatal("validation must not be cached")
	}
}

func TestProtocolFailures(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		want       error
	}{
		{"expired", `{"code":"TOKEN_EXPIRED"}`, 401, authn.ErrUnauthenticated},
		{"app secret invalid", `{"code":"APP_CREDENTIAL_INVALID"}`, 401, authn.ErrUnavailable},
		{"ambiguous unauthorized", `{"message":"bad token"}`, 401, authn.ErrUnavailable},
		{"denied", strings.Replace(validReply, `"allowed":true`, `"allowed":false`, 1), 200, authn.ErrForbidden},
		{"wrong app", strings.Replace(validReply, `"agent"`, `"another"`, 1), 200, authn.ErrUnavailable},
		{"wrong issuer", strings.Replace(validReply, `"paas"`, `"other"`, 1), 200, authn.ErrUnavailable},
		{"missing active", `{"code":0,"data":{}}`, 200, authn.ErrUnavailable},
		{"missing allowed", strings.Replace(validReply, `"allowed":true,`, ``, 1), 200, authn.ErrUnavailable},
		{"missing code", `{"data":{"active":true}}`, 200, authn.ErrUnavailable},
		{"invalid JSON", `not json user-token app-secret`, 200, authn.ErrUnavailable},
		{"past expiry", strings.Replace(validReply, `"active":true`, `"expiresAt":"2000-01-01T00:00:00Z","active":true`, 1), 200, authn.ErrUnauthenticated},
		{"upstream error", validReply, 503, authn.ErrUnavailable},
		{"non-200 success", validReply, 403, authn.ErrUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})
			_, err := c.Authenticate(context.Background(), "user-token")
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if strings.Contains(err.Error(), "user-token") || strings.Contains(err.Error(), "app-secret") {
				t.Fatal("credential leaked")
			}
		})
	}
}

func TestNoCredentialRedirectAndCancellation(t *testing.T) {
	var redirected atomic.Int32
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected.Add(1) }))
	defer target.Close()
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	})
	if _, err := c.Authenticate(context.Background(), "user-token"); !errors.Is(err, authn.ErrUnavailable) {
		t.Fatal(err)
	}
	if redirected.Load() != 0 {
		t.Fatal("followed credential-bearing redirect")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Authenticate(ctx, "user-token"); !errors.Is(err, authn.ErrUnavailable) {
		t.Fatal(err)
	}
}

func TestTimeoutAndRevocation(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(100 * time.Millisecond):
		}
	})
	c.httpClient.Timeout = 20 * time.Millisecond
	if _, err := c.Authenticate(context.Background(), "user-token"); !errors.Is(err, authn.ErrUnavailable) {
		t.Fatal(err)
	}
	for _, code := range []string{"0", `"TOKEN_REVOKED"`, `"TOKEN_EXPIRED"`} {
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/open/auth/logout" {
				t.Error("wrong revoke endpoint")
			}
			_, _ = w.Write([]byte(`{"code":` + code + `}`))
		})
		if err := c.Revoke(context.Background(), "user-token"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInvalidConfigAndEmptyToken(t *testing.T) {
	base := Config{BaseURL: "https://paas.example.com", IssuerID: "paas", AppCode: "agent"}
	for _, change := range []func(*Config){
		func(c *Config) { c.BaseURL = "http://paas.example.com" },
		func(c *Config) { c.BaseURL = "https://user:secret@paas.example.com" },
		func(c *Config) { c.AppCode = "" },
		func(c *Config) { c.IssuerID = "" },
		func(c *Config) { c.Endpoints.Check = "//other.example.com/check" },
		func(c *Config) { c.Endpoints.Check = "https://other.example.com/check" },
		func(c *Config) { c.Timeout = "-1s" },
	} {
		cfg := base
		change(&cfg)
		if _, err := newClient(cfg, "secret", nil); err == nil {
			t.Fatalf("accepted invalid config: %+v", cfg)
		}
	}
	t.Setenv("PAAS_TEST_SECRET", "")
	base.AppSecretEnv = "PAAS_TEST_SECRET"
	if _, err := New(base); err == nil {
		t.Fatal("accepted empty secret")
	}
	c := testClient(t, func(http.ResponseWriter, *http.Request) { t.Error("empty token reached PaaS") })
	if _, err := c.Authenticate(context.Background(), ""); !errors.Is(err, authn.ErrUnauthenticated) {
		t.Fatal(err)
	}
}
