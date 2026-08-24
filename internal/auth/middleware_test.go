package auth

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	hertzconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/route"

	"cerebro/internal/config"
	"cerebro/internal/domain"
)

const testSecret = "unit-test-secret"

// newBasicAuthSvc builds an enabled basic-auth service (u/p).
func newBasicAuthSvc(t *testing.T) domain.AuthService {
	t.Helper()
	svc, err := NewAuthService(config.AuthConfig{
		Type:     "basic",
		Settings: config.AuthSettings{Username: "u", Password: "p"},
	})
	if err != nil {
		t.Fatalf("NewAuthService: %v", err)
	}
	return svc
}

// newEngine registers the middleware in front of a probe handler that echoes
// the context user (mirroring how process() reads it).
func newEngine(svc domain.AuthService) *route.Engine {
	engine := route.NewEngine(hertzconfig.NewOptions(nil))
	engine.Use(AuthMiddleware(svc, testSecret))
	engine.POST("/probe", func(ctx context.Context, c *app.RequestContext) {
		name := ""
		if u, ok := c.Get("user"); ok {
			if user, ok := u.(*domain.User); ok && user != nil {
				name = user.Name
			}
		}
		c.JSON(200, map[string]string{"user": name})
	})
	return engine
}

func perform(engine *route.Engine, headers ...ut.Header) *ut.ResponseRecorder {
	return ut.PerformRequest(engine, "POST", "/probe", &ut.Body{
		Body: strings.NewReader(`{}`),
		Len:  2,
	}, headers...)
}

func TestAuthMiddleware_APICallWithoutSession_Returns303Envelope(t *testing.T) {
	engine := newEngine(newBasicAuthSvc(t))
	res := perform(engine).Result() // no Accept header -> API call

	if res.StatusCode() != 200 {
		t.Fatalf("HTTP status = %d, want 200 (envelope)", res.StatusCode())
	}
	var env domain.Response
	if err := json.Unmarshal(res.Body(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v (body %q)", err, res.Body())
	}
	if env.Status != 303 || string(env.Body) != "null" {
		t.Errorf("envelope = {%d, %s}, want {303, null}", env.Status, env.Body)
	}
}

func TestAuthMiddleware_BrowserWithoutSession_RedirectsToLogin(t *testing.T) {
	engine := newEngine(newBasicAuthSvc(t))
	res := perform(engine, ut.Header{Key: "Accept", Value: "text/html"}).Result()

	if res.StatusCode() != 302 {
		t.Fatalf("HTTP status = %d, want 302", res.StatusCode())
	}
	if loc := string(res.Header.Peek("Location")); loc != "/login" {
		t.Errorf("Location = %q, want /login", loc)
	}
	ck := &protocol.Cookie{}
	ck.SetKey(RedirectCookie)
	if !res.Header.Cookie(ck) {
		t.Fatalf("redirect cookie not set")
	}
	// Hertz percent-encodes cookie values on write; decode to compare.
	if v, err := url.QueryUnescape(string(ck.Value())); err != nil || v != "/probe" {
		t.Errorf("redirect cookie = %q (decoded %q, err %v), want /probe", ck.Value(), v, err)
	}
}

func TestAuthMiddleware_ValidSession_SetsUser(t *testing.T) {
	engine := newEngine(newBasicAuthSvc(t))

	// Build the cookie value via the same codec the login handler uses.
	sc := newSecureCookie(testSecret)
	encoded, err := sc.Encode(SessionCookie, "alice")
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	res := perform(engine, ut.Header{
		Key:   "Cookie",
		Value: SessionCookie + "=" + encoded,
	}).Result()
	if res.StatusCode() != 200 {
		t.Fatalf("HTTP status = %d, want 200", res.StatusCode())
	}
	if !strings.Contains(string(res.Body()), `"user":"alice"`) {
		t.Errorf("body = %s, want user alice injected", res.Body())
	}
}

func TestAuthMiddleware_TamperedSession_TreatedAsNoSession(t *testing.T) {
	engine := newEngine(newBasicAuthSvc(t))

	sc := newSecureCookie(testSecret)
	encoded, err := sc.Encode(SessionCookie, "alice")
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	tampered := encoded[:len(encoded)-2] + "zz" // break the HMAC

	res := perform(engine, ut.Header{
		Key:   "Cookie",
		Value: SessionCookie + "=" + tampered,
	}).Result()
	var env domain.Response
	if err := json.Unmarshal(res.Body(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v (body %q)", err, res.Body())
	}
	if env.Status != 303 {
		t.Errorf("envelope status = %d, want 303 (tampered cookie rejected)", env.Status)
	}
}

func TestAuthMiddleware_Disabled_PassesThroughWithoutUser(t *testing.T) {
	svc, err := NewAuthService(config.AuthConfig{}) // type "" -> NoneAuth
	if err != nil {
		t.Fatalf("NewAuthService: %v", err)
	}
	engine := newEngine(svc)

	res := perform(engine).Result()
	if res.StatusCode() != 200 {
		t.Fatalf("HTTP status = %d, want 200", res.StatusCode())
	}
	if !strings.Contains(string(res.Body()), `"user":""`) {
		t.Errorf("body = %s, want empty user (auth disabled, no 303)", res.Body())
	}
}

func TestSafeRedirectURL(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"/", "/"},
		{"/overview", "/overview"},
		{"/rest?x=1&y=2", "/rest?x=1&y=2"},
		{"", "/"},
		{"//evil.com", "/"},            // protocol-relative open redirect
		{"//evil.com/x", "/"},          // protocol-relative open redirect
		{"/\\evil.com", "/"},           // backslash variant some browsers normalize
		{"https://evil.com", "/"},      // absolute URL
		{"http://evil.com", "/"},       // absolute URL
		{"javascript:alert(1)", "/"},   // scheme injection
		{"/x\r\nSet-Cookie: a=b", "/"}, // header injection attempt
		{"/overview\x00", "/"},         // NUL
	}
	for _, tc := range cases {
		if got := SafeRedirectURL(tc.in); got != tc.want {
			t.Errorf("SafeRedirectURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSessionCookieRoundTrip(t *testing.T) {
	engine := route.NewEngine(hertzconfig.NewOptions(nil))
	var gotUser string
	var gotOK bool
	engine.GET("/who", func(ctx context.Context, c *app.RequestContext) {
		gotUser, gotOK = SessionUser(c, testSecret)
		c.Status(200)
	})
	engine.GET("/set", func(ctx context.Context, c *app.RequestContext) {
		if err := SetSessionCookie(c, testSecret, "bob"); err != nil {
			c.Status(500)
			return
		}
		c.Status(200)
	})

	// Issue the cookie.
	w := ut.PerformRequest(engine, "GET", "/set", nil)
	res := w.Result()
	if res.StatusCode() != 200 {
		t.Fatalf("set: HTTP status = %d, want 200", res.StatusCode())
	}
	ck := &protocol.Cookie{}
	ck.SetKey(SessionCookie)
	if !res.Header.Cookie(ck) {
		t.Fatalf("session cookie not set")
	}
	if !ck.HTTPOnly() {
		t.Errorf("session cookie must be HttpOnly")
	}

	// Read it back.
	ut.PerformRequest(engine, "GET", "/who", nil, ut.Header{
		Key:   "Cookie",
		Value: SessionCookie + "=" + string(ck.Value()),
	})
	if !gotOK || gotUser != "bob" {
		t.Errorf("SessionUser = (%q, %v), want (bob, true)", gotUser, gotOK)
	}
}

func TestSetSessionCookie_EmptySecretFailsClosed(t *testing.T) {
	engine := route.NewEngine(hertzconfig.NewOptions(nil))
	engine.GET("/set", func(ctx context.Context, c *app.RequestContext) {
		if err := SetSessionCookie(c, "", "bob"); err != nil {
			c.Status(500)
			return
		}
		c.Status(200)
	})
	res := ut.PerformRequest(engine, "GET", "/set", nil).Result()
	if res.StatusCode() != 500 {
		t.Errorf("HTTP status = %d, want 500 (no session issued without a secret)", res.StatusCode())
	}
	ck := &protocol.Cookie{}
	ck.SetKey(SessionCookie)
	if res.Header.Cookie(ck) && len(ck.Value()) > 0 {
		t.Errorf("session cookie must not be issued with an empty secret")
	}
}
