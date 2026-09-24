// Package paas implements the application-side PaaS token validation protocol.
package paas

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/xiehqing/hiauthx/authn"
)

type Endpoints struct {
	Authorize string `json:"authorize" yaml:"authorize" mapstructure:"authorize"`
	Exchange  string `json:"exchange" yaml:"exchange" mapstructure:"exchange"`
	Check     string `json:"check" yaml:"check" mapstructure:"check"`
	Logout    string `json:"logout" yaml:"logout" mapstructure:"logout"`
}

type Config struct {
	AllowInsecureHTTP bool      `json:"allowInsecureHttp" yaml:"allow-insecure-http" mapstructure:"allow-insecure-http"`
	CallbackURL       string    `json:"callbackUrl" yaml:"callback-url" mapstructure:"callback-url"`
	BaseURL           string    `json:"baseUrl" yaml:"base-url" mapstructure:"base-url"`
	IssuerID          string    `json:"issuerId" yaml:"issuer-id" mapstructure:"issuer-id"`
	AppCode           string    `json:"appCode" yaml:"app-code" mapstructure:"app-code"`
	AppSecretEnv      string    `json:"appSecretEnv" yaml:"app-secret-env" mapstructure:"app-secret-env"`
	AppSecret         string    `json:"appSecret" yaml:"app-secret" mapstructure:"app-secret"`
	Timeout           string    `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
	Endpoints         Endpoints `json:"endpoints" yaml:"endpoints" mapstructure:"endpoints"`
}

type Client struct {
	allowInsecureHTTP                            bool
	httpClient                                   *http.Client
	checkURL, logoutURL, issuer, appCode, secret string
	authorizeURL, exchangeURL                    string
}

// New reads the secret only on the server. HTTP requires explicit opt-in.
func New(cfg Config) (*Client, error) {
	secret := strings.TrimSpace(cfg.AppSecret)
	if secret == "" {
		secret, _ = os.LookupEnv(cfg.AppSecretEnv)
		secret = strings.TrimSpace(secret)
	}
	if secret == "" {
		return nil, errors.New("PaaS application secret environment variable is missing or empty")
	}
	return newClient(cfg, secret, nil)
}

func newClient(cfg Config, secret string, transport http.RoundTripper) (*Client, error) {
	base, err := url.Parse(cfg.BaseURL)
	if err != nil || !allowedScheme(base.Scheme, cfg.AllowInsecureHTTP) || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" || base.Opaque != "" {
		return nil, errors.New("PaaS base-url must be an absolute HTTPS URL (HTTP requires allow-insecure-http), without credentials, query or fragment")
	}
	if strings.TrimSpace(cfg.IssuerID) == "" || strings.TrimSpace(cfg.AppCode) == "" || strings.Contains(cfg.AppCode, ":") || strings.TrimSpace(secret) == "" {
		return nil, errors.New("PaaS issuer-id, app-code and application secret are required")
	}
	timeout := 3 * time.Second
	if cfg.Timeout != "" {
		timeout, err = time.ParseDuration(cfg.Timeout)
		if err != nil || timeout <= 0 || timeout > 30*time.Second {
			return nil, errors.New("PaaS timeout must be greater than zero and at most 30s")
		}
	}
	if cfg.Endpoints.Check == "" {
		cfg.Endpoints.Check = "/open/auth/check"
	}
	if cfg.Endpoints.Logout == "" {
		cfg.Endpoints.Logout = "/open/auth/logout"
	}
	endpoint := func(path string) (string, error) {
		u, err := url.Parse(path)
		if err != nil || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || u.IsAbs() || u.Host != "" || u.RawQuery != "" || u.Fragment != "" {
			return "", errors.New("PaaS endpoints must be absolute paths on the configured origin")
		}
		return base.ResolveReference(u).String(), nil
	}
	if cfg.Endpoints.Authorize == "" {
		cfg.Endpoints.Authorize = "/open/apps/authorize"
	}
	if cfg.Endpoints.Exchange == "" {
		cfg.Endpoints.Exchange = "/open/apps/tickets/exchange"
	}
	authorizeURL, err := endpoint(cfg.Endpoints.Authorize)
	if err != nil {
		return nil, err
	}
	exchangeURL, err := endpoint(cfg.Endpoints.Exchange)
	if err != nil {
		return nil, err
	}
	checkURL, err := endpoint(cfg.Endpoints.Check)
	if err != nil {
		return nil, err
	}
	logoutURL, err := endpoint(cfg.Endpoints.Logout)
	if err != nil {
		return nil, err
	}
	return &Client{
		allowInsecureHTTP: cfg.AllowInsecureHTTP,
		httpClient:        &http.Client{Timeout: timeout, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		checkURL:          checkURL, logoutURL: logoutURL, issuer: cfg.IssuerID, appCode: cfg.AppCode, secret: secret,
		authorizeURL: authorizeURL, exchangeURL: exchangeURL,
	}, nil
}

type checkResult struct {
	Active    *bool      `json:"active"`
	Allowed   *bool      `json:"allowed"`
	AppCode   string     `json:"appCode"`
	Issuer    string     `json:"issuer"`
	Subject   string     `json:"subject"`
	ExpiresAt *time.Time `json:"expiresAt"`
	User      struct {
		Username    string `json:"username"`
		DisplayName string `json:"displayName"`
	} `json:"user"`
}

func (c *Client) Authenticate(ctx context.Context, token string) (*authn.Identity, error) {
	if strings.TrimSpace(token) == "" {
		return nil, authn.ErrUnauthenticated
	}
	data, code, err := c.call(ctx, c.checkURL, token)
	if err != nil {
		return nil, err
	}
	if code != "0" {
		return nil, protocolError(code)
	}
	var result checkResult
	if json.Unmarshal(data, &result) != nil || result.Active == nil {
		return nil, authn.ErrUnavailable
	}
	if !*result.Active {
		return nil, authn.ErrUnauthenticated
	}
	if result.Allowed == nil || result.AppCode != c.appCode || result.Issuer != c.issuer || strings.TrimSpace(result.Subject) == "" {
		return nil, authn.ErrUnavailable
	}
	if result.ExpiresAt != nil && !result.ExpiresAt.After(time.Now()) {
		return nil, authn.ErrUnauthenticated
	}
	if !*result.Allowed {
		return nil, authn.ErrForbidden
	}
	return &authn.Identity{Issuer: result.Issuer, Subject: result.Subject, Username: result.User.Username, DisplayName: result.User.DisplayName, ExpiresAt: result.ExpiresAt}, nil
}

func (c *Client) Revoke(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return authn.ErrUnauthenticated
	}
	_, code, err := c.call(ctx, c.logoutURL, token)
	if err != nil {
		return err
	}
	if code == "0" || code == "TOKEN_INVALID" || code == "TOKEN_EXPIRED" || code == "TOKEN_REVOKED" {
		return nil
	}
	return protocolError(code)
}

// Responses use {code: 0, data: ...}. Named codes distinguish app credentials
// from user credentials; ambiguous HTTP errors fail as unavailable, not as logout.
func (c *Client) call(ctx context.Context, endpoint, token string) (json.RawMessage, string, error) {
	return c.callJSON(ctx, endpoint, map[string]string{"accessToken": token})
}

func (c *Client) callJSON(ctx context.Context, endpoint string, payload any) (json.RawMessage, string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", authn.ErrUnavailable
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, "", authn.ErrUnavailable
	}
	req.SetBasicAuth(c.appCode, c.secret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", authn.ErrUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode < 400 || resp.StatusCode >= 500 {
		return nil, "", authn.ErrUnavailable
	}
	const limit = 1 << 20
	raw, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || len(raw) > limit {
		return nil, "", authn.ErrUnavailable
	}
	var envelope struct {
		Code json.RawMessage `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(raw, &envelope) != nil || len(envelope.Code) == 0 {
		return nil, "", authn.ErrUnavailable
	}
	code := string(envelope.Code)
	if strings.HasPrefix(code, "\"") {
		if json.Unmarshal(envelope.Code, &code) != nil {
			return nil, "", authn.ErrUnavailable
		}
	}
	if resp.StatusCode != http.StatusOK && code == "0" {
		return nil, "", authn.ErrUnavailable
	}
	return envelope.Data, code, nil
}

func protocolError(code string) error {
	switch code {
	case "TOKEN_INVALID", "TOKEN_EXPIRED", "TOKEN_REVOKED", "USER_DISABLED":
		return authn.ErrUnauthenticated
	case "APP_ACCESS_DENIED":
		return authn.ErrForbidden
	default:
		return authn.ErrUnavailable
	}
}

// String deliberately excludes secrets and URLs that might contain credentials.
func (c *Client) String() string { return fmt.Sprintf("PaaSClient(%s)", c.appCode) }

func allowedScheme(scheme string, allowHTTP bool) bool {
	return scheme == "https" || (allowHTTP && scheme == "http")
}
