package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/sirupsen/logrus"

	"cerebro/internal/auth"
	"cerebro/internal/config"
	"cerebro/internal/domain"
	"cerebro/internal/handler"
	"cerebro/internal/service"
)

// fakeElastic satisfies the ElasticClient interface for the two methods the
// navbar exercises; the embedded interface panics on anything else.
type fakeElastic struct {
	domain.ElasticClient
}

func (f *fakeElastic) ClusterHealth(ctx context.Context, t domain.ElasticServer) (*domain.Response, error) {
	return &domain.Response{Status: 200, Body: json.RawMessage(`{"cluster_name":"c","status":"green"}`)}, nil
}

func (f *fakeElastic) BackendInfo(ctx context.Context, t domain.ElasticServer) (domain.BackendInfo, error) {
	return domain.BackendInfo{Type: "elasticsearch", Version: "9.4.3"}, nil
}

// newServerEngine returns handlers + config for NewServer with the given
// auth type, in a temp CWD that contains a minimal SPA shell so
// c.File("output/public/index.html") resolves.
func newServerEngine(t *testing.T, authType string) (*handler.Handlers, *config.Config) {
	t.Helper()

	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.MkdirAll(filepath.Join("output", "public"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join("output", "public", "index.html"), []byte("<html>spa</html>"), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}

	cfg := &config.Config{
		Secret:   "server-test-secret",
		BasePath: "/",
		Auth: config.AuthConfig{
			Type:     authType,
			Settings: config.AuthSettings{Username: "u", Password: "p"},
		},
	}
	authSvc, err := auth.NewAuthService(cfg.Auth)
	if err != nil {
		t.Fatalf("NewAuthService: %v", err)
	}
	hosts, err := config.NewHosts([]config.HostConfig{
		{Host: "http://es:9200", Name: "test"},
	})
	if err != nil {
		t.Fatalf("NewHosts: %v", err)
	}
	h := &handler.Handlers{
		Auth:   authSvc,
		Hosts:  hosts,
		Navbar: service.NewNavbarService(&fakeElastic{}),
		Log:    logrus.New(),
		Secret: cfg.Secret,
	}
	return h, cfg
}

func envelope(t *testing.T, body []byte) domain.Response {
	t.Helper()
	var env domain.Response
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v (body %q)", err, body)
	}
	return env
}

func sessionCookieValue(t *testing.T, res *protocol.Response) (string, bool) {
	t.Helper()
	ck := &protocol.Cookie{}
	ck.SetKey(auth.SessionCookie)
	if !res.Header.Cookie(ck) {
		return "", false
	}
	return string(ck.Value()), true
}

func TestServerAuthEnabled_APIBlockedWithoutSession(t *testing.T) {
	h, cfg := newServerEngine(t, "basic")
	srv := NewServer(0, cfg, h)

	w := ut.PerformRequest(srv.Engine, "POST", "/navbar", &ut.Body{
		Body: strings.NewReader(`{"host":"test"}`),
		Len:  len(`{"host":"test"}`),
	})
	res := w.Result()
	if res.StatusCode() != http.StatusOK {
		t.Fatalf("HTTP status = %d, want 200 (envelope)", res.StatusCode())
	}
	env := envelope(t, res.Body())
	if env.Status != 303 || string(env.Body) != "null" {
		t.Errorf("envelope = {%d, %s}, want {303, null}", env.Status, env.Body)
	}
}

func TestServerAuthEnabled_BrowserNavigationRedirectsToLogin(t *testing.T) {
	h, cfg := newServerEngine(t, "basic")
	srv := NewServer(0, cfg, h)

	// Deep link (SPA fallback) without a session -> 302 /login + redirect cookie.
	w := ut.PerformRequest(srv.Engine, "GET", "/overview", nil,
		ut.Header{Key: "Accept", Value: "text/html"})
	res := w.Result()
	if res.StatusCode() != http.StatusFound {
		t.Fatalf("HTTP status = %d, want 302", res.StatusCode())
	}
	if loc := string(res.Header.Peek("Location")); loc != "/login" {
		t.Errorf("Location = %q, want /login", loc)
	}
	ck := &protocol.Cookie{}
	ck.SetKey(auth.RedirectCookie)
	if !res.Header.Cookie(ck) {
		t.Fatalf("redirect cookie not set")
	}
	// Hertz percent-encodes cookie values; browsers echo them back verbatim.
	v, err := url.QueryUnescape(string(ck.Value()))
	if err != nil {
		t.Fatalf("redirect cookie not percent-encoded: %q", ck.Value())
	}
	if v != "/overview" {
		t.Errorf("redirect cookie = %q, want /overview", v)
	}

	// GET / without a session -> 302 /login as well.
	w = ut.PerformRequest(srv.Engine, "GET", "/", nil,
		ut.Header{Key: "Accept", Value: "text/html"})
	res = w.Result()
	if res.StatusCode() != http.StatusFound {
		t.Errorf("GET / HTTP status = %d, want 302", res.StatusCode())
	}
}

func TestServerAuthEnabled_PublicRoutesReachableWithoutSession(t *testing.T) {
	h, cfg := newServerEngine(t, "basic")
	srv := NewServer(0, cfg, h)

	// Hosts list is pre-auth (connect screen).
	w := ut.PerformRequest(srv.Engine, "GET", "/connect/hosts", nil)
	res := w.Result()
	if res.StatusCode() != http.StatusOK {
		t.Errorf("GET /connect/hosts HTTP status = %d, want 200", res.StatusCode())
	}
	if env := envelope(t, res.Body()); env.Status != 200 {
		t.Errorf("hosts envelope status = %d, want 200", env.Status)
	}

	// Login page renders the SPA shell (no auth redirect).
	w = ut.PerformRequest(srv.Engine, "GET", "/login", nil,
		ut.Header{Key: "Accept", Value: "text/html"})
	res = w.Result()
	if res.StatusCode() != http.StatusOK {
		t.Errorf("GET /login HTTP status = %d, want 200", res.StatusCode())
	}
	if !strings.Contains(string(res.Body()), "spa") {
		t.Errorf("GET /login body = %q, want the SPA shell", res.Body())
	}

	// Favicon stays reachable.
	w = ut.PerformRequest(srv.Engine, "GET", "/favicon.ico", nil)
	if res = w.Result(); res.StatusCode() != http.StatusNoContent {
		t.Errorf("GET /favicon.ico HTTP status = %d, want 204", res.StatusCode())
	}

	// Logout is callable without a session.
	w = ut.PerformRequest(srv.Engine, "POST", "/auth/logout", nil)
	res = w.Result()
	if res.StatusCode() != http.StatusFound {
		t.Errorf("POST /auth/logout HTTP status = %d, want 302", res.StatusCode())
	}
}

func TestServerAuthEnabled_LoginFlow(t *testing.T) {
	h, cfg := newServerEngine(t, "basic")
	srv := NewServer(0, cfg, h)

	form := "user=u&password=p"
	w := ut.PerformRequest(srv.Engine, "POST", "/auth/login", &ut.Body{
		Body: strings.NewReader(form),
		Len:  len(form),
	}, ut.Header{Key: "Content-Type", Value: "application/x-www-form-urlencoded"})
	res := w.Result()
	if res.StatusCode() != http.StatusFound {
		t.Fatalf("login HTTP status = %d, want 302", res.StatusCode())
	}
	cookie, ok := sessionCookieValue(t, res)
	if !ok || cookie == "" {
		t.Fatalf("session cookie not set after successful login")
	}

	// The session cookie unblocks the API and injects the username.
	w = ut.PerformRequest(srv.Engine, "POST", "/navbar", &ut.Body{
		Body: strings.NewReader(`{"host":"test"}`),
		Len:  len(`{"host":"test"}`),
	}, ut.Header{Key: "Cookie", Value: auth.SessionCookie + "=" + cookie})
	res = w.Result()
	if res.StatusCode() != http.StatusOK {
		t.Fatalf("navbar HTTP status = %d, want 200", res.StatusCode())
	}
	env := envelope(t, res.Body())
	if env.Status != 200 {
		t.Fatalf("navbar envelope status = %d, want 200", env.Status)
	}
	if !strings.Contains(string(env.Body), `"username":"u"`) {
		t.Errorf("navbar body = %s, want username injected", env.Body)
	}

	// Wrong credentials: no session cookie, back to /login.
	bad := "user=u&password=wrong"
	w = ut.PerformRequest(srv.Engine, "POST", "/auth/login", &ut.Body{
		Body: strings.NewReader(bad),
		Len:  len(bad),
	}, ut.Header{Key: "Content-Type", Value: "application/x-www-form-urlencoded"})
	res = w.Result()
	if res.StatusCode() != http.StatusFound {
		t.Fatalf("bad login HTTP status = %d, want 302", res.StatusCode())
	}
	if loc := string(res.Header.Peek("Location")); loc != "/login" {
		t.Errorf("bad login Location = %q, want /login", loc)
	}
	if _, ok := sessionCookieValue(t, res); ok {
		t.Errorf("session cookie must not be set on failed login")
	}
}

func TestServerAuthEnabled_PostLoginOpenRedirectBlocked(t *testing.T) {
	h, cfg := newServerEngine(t, "basic")
	srv := NewServer(0, cfg, h)

	form := "user=u&password=p"
	w := ut.PerformRequest(srv.Engine, "POST", "/auth/login", &ut.Body{
		Body: strings.NewReader(form),
		Len:  len(form),
	},
		ut.Header{Key: "Content-Type", Value: "application/x-www-form-urlencoded"},
		// Attacker-influenceable redirect cookie (protocol-relative URL).
		ut.Header{Key: "Cookie", Value: auth.RedirectCookie + "=//evil.com"},
	)
	res := w.Result()
	if res.StatusCode() != http.StatusFound {
		t.Fatalf("login HTTP status = %d, want 302", res.StatusCode())
	}
	if loc := string(res.Header.Peek("Location")); loc != "/" {
		t.Errorf("post-login Location = %q, want / (open redirect blocked)", loc)
	}
}

func TestServerAuthDisabled_BehaviorUnchanged(t *testing.T) {
	h, cfg := newServerEngine(t, "")
	srv := NewServer(0, cfg, h)

	// API routes work without any session and without the 303 envelope.
	w := ut.PerformRequest(srv.Engine, "POST", "/navbar", &ut.Body{
		Body: strings.NewReader(`{"host":"test"}`),
		Len:  len(`{"host":"test"}`),
	})
	res := w.Result()
	if res.StatusCode() != http.StatusOK {
		t.Fatalf("HTTP status = %d, want 200", res.StatusCode())
	}
	env := envelope(t, res.Body())
	if env.Status != 200 {
		t.Errorf("envelope status = %d, want 200 (auth disabled)", env.Status)
	}
	if strings.Contains(string(env.Body), "username") {
		t.Errorf("navbar body = %s, want no username when auth disabled", env.Body)
	}

	// GET / serves the SPA directly (no redirect).
	w = ut.PerformRequest(srv.Engine, "GET", "/", nil,
		ut.Header{Key: "Accept", Value: "text/html"})
	res = w.Result()
	if res.StatusCode() != http.StatusOK {
		t.Errorf("GET / HTTP status = %d, want 200", res.StatusCode())
	}
}
