package paas

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/xiehqing/hiauthx/authn"
	"github.com/xiehqing/infra/pkg/hertzx"
)

var ErrLoginRecordMissing = errors.New("login transaction missing or expired")

// LoginStore stores encrypted, short-lived transactions, never a login session.
// Take MUST atomically remove and return an unexpired record across all replicas.
type LoginStore interface {
	Put(ctx context.Context, key string, ciphertext []byte, expiresAt time.Time) error
	Take(ctx context.Context, key string) ([]byte, error)
}

type LoginFlow struct {
	secureCookie                     bool
	client                           *Client
	store                            LoginStore
	callback, origin, prefix, cookie string
	aead                             cipher.AEAD
}

type loginRecord struct {
	Verifier  string    `json:"verifier,omitempty"`
	ReturnTo  string    `json:"returnTo"`
	Token     string    `json:"token,omitempty"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// NewLoginFlow requires a same-origin frontend and registered callback.
// HTTPS is the default; HTTP requires the client configuration opt-in.
// The application secret derives a key for encrypted database handoff records.
func NewLoginFlow(client *Client, store LoginStore, callback string) (*LoginFlow, error) {
	if client == nil || store == nil {
		return nil, errors.New("PaaS login requires client and transaction store")
	}
	u, err := url.Parse(callback)
	if err != nil || !allowedScheme(u.Scheme, client.allowInsecureHTTP) || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" || !strings.HasSuffix(u.Path, "/auth/paas/callback") {
		return nil, errors.New("PaaS callback-url must use HTTPS (HTTP requires allow-insecure-http) and end in /auth/paas/callback, without credentials, query or fragment")
	}
	if strings.Contains(u.Path, "//") || strings.Contains(u.Path, "\\") || strings.Contains(u.Path, "..") {
		return nil, errors.New("invalid PaaS callback path")
	}
	mac := hmac.New(sha256.New, []byte(client.secret))
	_, _ = mac.Write([]byte("hiauthx-login-v1:" + client.appCode))
	block, err := aes.NewCipher(mac.Sum(nil))
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256([]byte(client.appCode))
	host := strings.ToLower(u.Hostname())
	defaultPort := "443"
	secure := u.Scheme == "https"
	cookiePrefix := "__Host-hiauthx-"
	if !secure {
		defaultPort = "80"
		cookiePrefix = "hiauthx-"
	}
	if port := u.Port(); port != "" && port != defaultPort {
		host = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return &LoginFlow{client: client, store: store, callback: callback, origin: u.Scheme + "://" + host, prefix: strings.TrimSuffix(u.Path, "/callback"), secureCookie: secure, cookie: cookiePrefix + hex.EncodeToString(hash[:6]), aead: aead}, nil
}

func (f *LoginFlow) LoginURL() string { return f.prefix + "/login" }

func securityHeaders(c *app.RequestContext) {
	c.Response.Header.Set("Cache-Control", "no-store")
	c.Response.Header.Set("Pragma", "no-cache")
	c.Response.Header.Set("Referrer-Policy", "no-referrer")
	c.Response.Header.Set("X-Content-Type-Options", "nosniff")
}

func randomValue() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func recordKey(kind, value string) string {
	hash := sha256.Sum256([]byte(kind + ":" + value))
	return hex.EncodeToString(hash[:])
}

func validRandom(value string) bool {
	b, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil && len(b) == 32
}

// SafeReturnTo only permits local application paths, including after URL decoding.
func SafeReturnTo(value string) bool {
	if value == "" || len(value) > 2048 || !strings.HasPrefix(value, "/") {
		return false
	}
	u, err := url.Parse(value)
	if err != nil || u.IsAbs() || u.Host != "" || u.Opaque != "" {
		return false
	}
	decoded, err := url.PathUnescape(value)
	if err != nil || strings.HasPrefix(decoded, "//") || strings.Contains(decoded, "\\") || strings.ContainsFunc(decoded, unicode.IsControl) {
		return false
	}
	// Returning to a login/callback endpoint would create a redirect loop.
	return !strings.Contains(u.Path, "/auth/paas/")
}

func (f *LoginFlow) put(ctx context.Context, key string, record loginRecord) error {
	plain, err := json.Marshal(record)
	if err != nil {
		return err
	}
	nonce := make([]byte, f.aead.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return err
	}
	encrypted := f.aead.Seal(nonce, nonce, plain, []byte(key))
	return f.store.Put(ctx, key, encrypted, record.ExpiresAt)
}

func (f *LoginFlow) take(ctx context.Context, key string) (loginRecord, error) {
	var record loginRecord
	encrypted, err := f.store.Take(ctx, key)
	if err != nil {
		return record, err
	}
	n := f.aead.NonceSize()
	if len(encrypted) < n {
		return record, authn.ErrUnavailable
	}
	plain, err := f.aead.Open(nil, encrypted[:n], encrypted[n:], []byte(key))
	if err != nil {
		return record, authn.ErrUnavailable
	}
	if json.Unmarshal(plain, &record) != nil {
		return record, authn.ErrUnavailable
	}
	if !record.ExpiresAt.After(time.Now()) {
		return record, ErrLoginRecordMissing
	}
	return record, nil
}

func (f *LoginFlow) setCookie(c *app.RequestContext, suffix, value string, age int) {
	c.SetCookie(f.cookie+suffix, value, age, "/", "", protocol.CookieSameSiteLaxMode, f.secureCookie, true)
}

func (f *LoginFlow) fail(c *app.RequestContext, reason string) {
	f.setCookie(c, "-login", "", -1)
	f.setCookie(c, "-handoff", "", -1)
	// Only fixed error codes enter the URL. Token, ticket and upstream errors never do.
	c.Redirect(http.StatusSeeOther, []byte(f.origin+"/?paas_error="+reason))
}

func (f *LoginFlow) Start(ctx context.Context, c *app.RequestContext) {
	securityHeaders(c)
	returnTo := c.DefaultQuery("return_to", "/")
	if !SafeReturnTo(returnTo) {
		hertzx.Abort(c, 400, "返回路径不合法")
		return
	}
	state, e1 := randomValue()
	binding, e2 := randomValue()
	verifier, e3 := randomValue()
	if e1 != nil || e2 != nil || e3 != nil {
		hertzx.Abort(c, 503, "暂时无法发起登录")
		return
	}
	err := f.put(ctx, recordKey("login", state+":"+binding), loginRecord{Verifier: verifier, ReturnTo: returnTo, ExpiresAt: time.Now().Add(5 * time.Minute)})
	if err != nil {
		hertzx.Abort(c, 503, "暂时无法发起登录")
		return
	}
	f.setCookie(c, "-login", binding, 300)
	f.setCookie(c, "-handoff", "", -1)
	challenge := sha256.Sum256([]byte(verifier))
	u, _ := url.Parse(f.client.authorizeURL)
	q := u.Query()
	q.Set("app_code", f.client.appCode)
	q.Set("redirect_uri", f.callback)
	q.Set("state", state)
	q.Set("code_challenge", base64.RawURLEncoding.EncodeToString(challenge[:]))
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, []byte(u.String()))
}

func (f *LoginFlow) Callback(ctx context.Context, c *app.RequestContext) {
	securityHeaders(c)
	state := c.Query("state")
	binding := string(c.Cookie(f.cookie + "-login"))
	ticket := c.Query("ticket")
	if !validRandom(state) || !validRandom(binding) || ticket == "" || len(ticket) > 4096 {
		f.fail(c, "invalid")
		return
	}
	record, err := f.take(ctx, recordKey("login", state+":"+binding))
	if err != nil {
		if errors.Is(err, ErrLoginRecordMissing) {
			f.fail(c, "invalid")
		} else {
			f.fail(c, "unavailable")
		}
		return
	}
	f.setCookie(c, "-login", "", -1)
	token, err := f.client.Exchange(ctx, ticket, f.callback, record.Verifier)
	if err != nil {
		reason := "unavailable"
		if errors.Is(err, authn.ErrUnauthenticated) {
			reason = "expired"
		}
		if errors.Is(err, authn.ErrForbidden) {
			reason = "denied"
		}
		f.fail(c, reason)
		return
	}
	handoff, err := randomValue()
	if err != nil {
		f.fail(c, "unavailable")
		return
	}
	err = f.put(ctx, recordKey("handoff", handoff), loginRecord{Token: token, ReturnTo: record.ReturnTo, ExpiresAt: time.Now().Add(30 * time.Second)})
	if err != nil {
		f.fail(c, "unavailable")
		return
	}
	f.setCookie(c, "-handoff", handoff, 30)
	c.Redirect(http.StatusSeeOther, []byte(f.origin+"/?paas_login=1"))
}

func (f *LoginFlow) Bootstrap(ctx context.Context, c *app.RequestContext) {
	securityHeaders(c)
	// SameSite alone is insufficient for sibling origins. The browser must send
	// the exact configured Origin and a custom same-origin request header.
	if string(c.GetHeader("Origin")) != f.origin || string(c.GetHeader("X-PaaS-Bootstrap")) != "1" {
		hertzx.Abort(c, 403, "登录交接来源不合法")
		return
	}
	binding := string(c.Cookie(f.cookie + "-handoff"))
	if !validRandom(binding) {
		hertzx.Abort(c, 401, "登录交接已失效，请重新登录")
		return
	}
	record, err := f.take(ctx, recordKey("handoff", binding))
	f.setCookie(c, "-handoff", "", -1)
	if err != nil {
		if errors.Is(err, ErrLoginRecordMissing) {
			hertzx.Abort(c, 401, "登录交接已失效，请重新登录")
		} else {
			hertzx.Abort(c, 503, "登录交接服务暂不可用")
		}
		return
	}
	// Revocation between callback and bootstrap must still prevent login.
	if _, err = f.client.Authenticate(ctx, record.Token); err != nil {
		status := 503
		if errors.Is(err, authn.ErrUnauthenticated) {
			status = 401
		}
		if errors.Is(err, authn.ErrForbidden) {
			status = 403
		}
		hertzx.Abort(c, status, "暂时无法完成登录，请重新发起")
		return
	}
	hertzx.Data(c, map[string]string{"accessToken": record.Token, "returnTo": record.ReturnTo})
}

func (c *Client) Exchange(ctx context.Context, ticket, callback, verifier string) (string, error) {
	if ticket == "" || verifier == "" {
		return "", authn.ErrUnauthenticated
	}
	data, code, err := c.callJSON(ctx, c.exchangeURL, map[string]string{"ticket": ticket, "callbackUrl": callback, "codeVerifier": verifier})
	if err != nil {
		return "", err
	}
	if code != "0" {
		switch code {
		case "TICKET_INVALID", "TICKET_EXPIRED", "TICKET_USED":
			return "", authn.ErrUnauthenticated
		}
		return "", protocolError(code)
	}
	var result struct {
		AccessToken string `json:"accessToken"`
	}
	if json.Unmarshal(data, &result) != nil || strings.TrimSpace(result.AccessToken) == "" || len(result.AccessToken) > 16384 {
		return "", authn.ErrUnavailable
	}
	if _, err := c.Authenticate(ctx, result.AccessToken); err != nil {
		return "", err
	}
	return result.AccessToken, nil
}
