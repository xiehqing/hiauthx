package paas

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
)

type testLoginStore struct {
	mu   sync.Mutex
	data map[string]struct {
		value  []byte
		expiry time.Time
	}
}

func newTestStore() *testLoginStore {
	return &testLoginStore{data: map[string]struct {
		value  []byte
		expiry time.Time
	}{}}
}
func (s *testLoginStore) Put(_ context.Context, key string, value []byte, expiry time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = struct {
		value  []byte
		expiry time.Time
	}{append([]byte(nil), value...), expiry}
	return nil
}
func (s *testLoginStore) Take(_ context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.data[key]
	delete(s.data, key)
	if !ok || !row.expiry.After(time.Now()) {
		return nil, ErrLoginRecordMissing
	}
	return row.value, nil
}

type flowHarness struct {
	flow      *LoginFlow
	store     *testLoginStore
	app       *server.Hertz
	challenge string
	exchanges atomic.Int32
	revoked   atomic.Bool
}

func newHarness(t *testing.T) *flowHarness {
	return newSchemeHarness(t, false, "https://app.example/api/v1/auth/paas/callback")
}

func newSchemeHarness(t *testing.T, insecure bool, callbackURL string) *flowHarness {
	t.Helper()
	h := &flowHarness{store: newTestStore()}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app, secret, ok := r.BasicAuth()
		if !ok || app != "agent" || secret != "secret" {
			t.Error("missing backend app credentials")
		}
		switch r.URL.Path {
		case "/open/apps/tickets/exchange":
			var in map[string]string
			_ = json.NewDecoder(r.Body).Decode(&in)
			hash := sha256.Sum256([]byte(in["codeVerifier"]))
			if in["ticket"] != "one-ticket" || in["callbackUrl"] != callbackURL || base64.RawURLEncoding.EncodeToString(hash[:]) != h.challenge {
				t.Error("incorrect exchange binding")
			}
			if h.exchanges.Add(1) > 1 {
				_, _ = w.Write([]byte(`{"code":"TICKET_USED"}`))
				return
			}
			_, _ = w.Write([]byte(`{"code":0,"data":{"accessToken":"original-paas-token"}}`))
		case "/open/auth/check":
			if h.revoked.Load() {
				_, _ = w.Write([]byte(`{"code":0,"data":{"active":false}}`))
				return
			}
			_, _ = w.Write([]byte(validReply))
		default:
			t.Error("unexpected upstream endpoint")
			w.WriteHeader(404)
		}
	})
	var s *httptest.Server
	if insecure {
		s = httptest.NewServer(handler)
	} else {
		s = httptest.NewTLSServer(handler)
	}
	t.Cleanup(s.Close)
	client, err := newClient(Config{AllowInsecureHTTP: insecure, BaseURL: s.URL, IssuerID: "paas", AppCode: "agent"}, "secret", s.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	h.flow, err = NewLoginFlow(client, h.store, callbackURL)
	if err != nil {
		t.Fatal(err)
	}
	h.app = server.New()
	h.app.GET("/api/v1/auth/paas/login", h.flow.Start)
	h.app.GET("/api/v1/auth/paas/callback", h.flow.Callback)
	h.app.POST("/api/v1/auth/paas/bootstrap", h.flow.Bootstrap)
	return h
}

func cookieValue(t *testing.T, r *ut.ResponseRecorder, name string) string {
	t.Helper()
	var cookies []string
	r.Header().VisitAllCookie(func(_ []byte, value []byte) { cookies = append(cookies, string(value)) })
	for _, header := range cookies {
		if strings.HasPrefix(header, name+"=") && !strings.HasPrefix(header, name+"=;") {
			if strings.Contains(strings.ToLower(header), "secure") != strings.HasPrefix(name, "__Host-") || !strings.Contains(strings.ToLower(header), "httponly") || !strings.Contains(strings.ToLower(header), "samesite=lax") {
				t.Fatal("unsafe transaction cookie", header)
			}
			return strings.SplitN(header, ";", 2)[0]
		}
	}
	t.Fatal("missing cookie", name, cookies)
	return ""
}

func (h *flowHarness) start(t *testing.T) (string, string) {
	t.Helper()
	r := ut.PerformRequest(h.app.Engine, "GET", "/api/v1/auth/paas/login?return_to=%2Fagents%3Ftab%3Dmine", nil)
	if r.Code != 302 {
		t.Fatal(r.Code, r.Body.String())
	}
	u, err := url.Parse(string(r.Header().Peek("Location")))
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("app_code") != "agent" || u.Query().Get("code_challenge_method") != "S256" {
		t.Fatal(u)
	}
	h.challenge = u.Query().Get("code_challenge")
	return u.Query().Get("state"), cookieValue(t, r, h.flow.cookie+"-login")
}

func (h *flowHarness) callback(t *testing.T, state, cookie string) *ut.ResponseRecorder {
	t.Helper()
	return ut.PerformRequest(h.app.Engine, "GET", "/api/v1/auth/paas/callback?ticket=one-ticket&state="+url.QueryEscape(state), nil, ut.Header{Key: "Cookie", Value: cookie})
}
func (h *flowHarness) bootstrap(cookie, origin string) *ut.ResponseRecorder {
	return ut.PerformRequest(h.app.Engine, "POST", "/api/v1/auth/paas/bootstrap", nil, ut.Header{Key: "Cookie", Value: cookie}, ut.Header{Key: "Origin", Value: origin}, ut.Header{Key: "X-PaaS-Bootstrap", Value: "1"})
}

func TestBrowserFlowReturnsSameTokenOnceWithoutURLOrStoreLeak(t *testing.T) {
	h := newHarness(t)
	state, cookie := h.start(t)
	callback := h.callback(t, state, cookie)
	location := string(callback.Header().Peek("Location"))
	if callback.Code != 303 || location != "https://app.example/?paas_login=1" {
		t.Fatal(callback.Code, location)
	}
	handoff := cookieValue(t, callback, h.flow.cookie+"-handoff")
	for _, row := range h.store.data {
		if bytes.Contains(row.value, []byte("original-paas-token")) {
			t.Fatal("plaintext token in shared store")
		}
	}
	if callback.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("cacheable callback")
	}
	response := h.bootstrap(handoff, "https://app.example")
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	var out struct {
		Data struct {
			Token    string `json:"accessToken"`
			ReturnTo string `json:"returnTo"`
		} `json:"data"`
	}
	if json.Unmarshal(response.Body.Bytes(), &out) != nil || out.Data.Token != "original-paas-token" || out.Data.ReturnTo != "/agents?tab=mine" {
		t.Fatal(response.Body.String())
	}
	if r := h.bootstrap(handoff, "https://app.example"); r.Code != 401 {
		t.Fatal("bootstrap replay", r.Code)
	}
	if r := h.callback(t, state, cookie); !strings.Contains(string(r.Header().Peek("Location")), "paas_error=invalid") {
		t.Fatal("callback replay accepted")
	}
	if h.exchanges.Load() != 1 {
		t.Fatal("ticket exchanged more than once")
	}
}

func TestWrongBrowserAndCrossOriginCannotConsumeValidFlow(t *testing.T) {
	h := newHarness(t)
	state, cookie := h.start(t)
	wrong, _ := randomValue()
	r := h.callback(t, state, h.flow.cookie+"-login="+wrong)
	if h.exchanges.Load() != 0 || !strings.Contains(string(r.Header().Peek("Location")), "paas_error=invalid") {
		t.Fatal("wrong browser accepted")
	}
	r = h.callback(t, state, cookie)
	handoff := cookieValue(t, r, h.flow.cookie+"-handoff")
	if r = h.bootstrap(handoff, "https://sibling.example"); r.Code != 403 {
		t.Fatal("cross-origin bootstrap accepted")
	}
	if r = h.bootstrap(handoff, "https://app.example"); r.Code != 200 {
		t.Fatal("cross-origin request consumed valid record")
	}
}

func TestRevocationBetweenCallbackAndBootstrap(t *testing.T) {
	h := newHarness(t)
	state, cookie := h.start(t)
	r := h.callback(t, state, cookie)
	handoff := cookieValue(t, r, h.flow.cookie+"-handoff")
	h.revoked.Store(true)
	r = h.bootstrap(handoff, "https://app.example")
	if r.Code != 401 || strings.Contains(r.Body.String(), "original-paas-token") {
		t.Fatal("revoked token delivered")
	}
}

func TestOnlyOneConcurrentBootstrapWins(t *testing.T) {
	h := newHarness(t)
	state, cookie := h.start(t)
	r := h.callback(t, state, cookie)
	handoff := cookieValue(t, r, h.flow.cookie+"-handoff")
	var wg sync.WaitGroup
	var won atomic.Int32
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if h.bootstrap(handoff, "https://app.example").Code == 200 {
				won.Add(1)
			}
		}()
	}
	wg.Wait()
	if won.Load() != 1 {
		t.Fatalf("%d concurrent bootstrap successes", won.Load())
	}
}

func TestExpiredOrTamperedRecordsFailClosed(t *testing.T) {
	for _, tamper := range []bool{false, true} {
		h := newHarness(t)
		state, cookie := h.start(t)
		for key, row := range h.store.data {
			if tamper {
				row.value[0] ^= 1
			} else {
				row.expiry = time.Now().Add(-time.Second)
			}
			h.store.data[key] = row
		}
		r := h.callback(t, state, cookie)
		if h.exchanges.Load() != 0 || !strings.Contains(string(r.Header().Peek("Location")), "paas_error=") {
			t.Fatal("invalid transaction accepted")
		}
	}
}

func TestReturnPathsAndCallbackValidation(t *testing.T) {
	for _, path := range []string{"https://evil.example", "//evil.example", "/%2Fevil.example", "/\\evil.example", "/%5Cevil.example", "/%0d%0aLocation:x", "/api/v1/auth/paas/login"} {
		if SafeReturnTo(path) {
			t.Fatal("unsafe return path", path)
		}
	}
	if !SafeReturnTo("/agents?filter=mine#details") {
		t.Fatal("valid route rejected")
	}
	h := newHarness(t)
	for _, callback := range []string{"http://app.example/api/v1/auth/paas/callback", "https://user:secret@app.example/api/v1/auth/paas/callback", "https://app.example/api/v1/auth/paas/callback?token=bad"} {
		if _, err := NewLoginFlow(h.flow.client, h.store, callback); err == nil {
			t.Fatal("unsafe callback accepted")
		}
	}
}

func TestHTTPOptInBrowserFlow(t *testing.T) {
	h := newSchemeHarness(t, true, "http://192.168.1.20:8080/api/v1/auth/paas/callback")
	state, cookie := h.start(t)
	if strings.HasPrefix(h.flow.cookie, "__Host-") {
		t.Fatal("HTTP cookie has secure-only prefix")
	}
	callback := h.callback(t, state, cookie)
	if callback.Code != 303 || string(callback.Header().Peek("Location")) != "http://192.168.1.20:8080/?paas_login=1" {
		t.Fatal("incorrect HTTP redirect")
	}
	handoff := cookieValue(t, callback, h.flow.cookie+"-handoff")
	for _, origin := range []string{"https://192.168.1.20:8080", "http://192.168.1.20", "http://other.internal:8080"} {
		if r := h.bootstrap(handoff, origin); r.Code != 403 {
			t.Fatal("wrong origin accepted", origin)
		}
	}
	if r := h.bootstrap(handoff, h.flow.origin); r.Code != 200 || !strings.Contains(r.Body.String(), "original-paas-token") {
		t.Fatal(r.Code, r.Body.String())
	}
	if r := h.bootstrap(handoff, h.flow.origin); r.Code != 401 {
		t.Fatal("HTTP handoff replay accepted")
	}
}

func TestHTTPConfigurationAndCookieScheme(t *testing.T) {
	for _, allow := range []bool{false, true} {
		for _, scheme := range []string{"http", "https", "ftp"} {
			client, err := newClient(Config{BaseURL: scheme + "://paas.internal", AllowInsecureHTTP: allow, IssuerID: "paas", AppCode: "agent"}, "secret", nil)
			want := scheme == "https" || (scheme == "http" && allow)
			if (err == nil) != want {
				t.Fatal("incorrect scheme validation", scheme, allow, err)
			}
			if err != nil {
				continue
			}
			for _, callback := range []struct {
				url, origin string
				secure      bool
			}{
				{"https://APP.internal:443/api/v1/auth/paas/callback", "https://app.internal", true},
				{"http://APP.internal:80/api/v1/auth/paas/callback", "http://app.internal", false},
				{"http://[::1]:8080/api/v1/auth/paas/callback", "http://[::1]:8080", false},
			} {
				flow, err := NewLoginFlow(client, newTestStore(), callback.url)
				if !callback.secure && !allow {
					if err == nil {
						t.Fatal("HTTP callback accepted without opt-in")
					}
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				if flow.origin != callback.origin || flow.secureCookie != callback.secure {
					t.Fatal("incorrect origin or cookie security", flow.origin)
				}
				app := server.New()
				app.GET("/login", flow.Start)
				response := ut.PerformRequest(app.Engine, "GET", "/login", nil)
				cookieValue(t, response, flow.cookie+"-login")
			}
		}
	}
}
