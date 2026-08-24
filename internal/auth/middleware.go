package auth

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/gorilla/securecookie"

	"cerebro/internal/domain"
)

const SessionCookie = "cerebro_session"
const RedirectCookie = "redirect"

// sessionMaxAge bounds both the browser cookie lifetime and the timestamp
// embedded by securecookie that Decode accepts.
const sessionMaxAge = 86400

// newSecureCookie builds the session cookie codec. The value is signed
// (HMAC-SHA256) with secret but NOT encrypted (nil block key): the cookie
// only carries the username, so a tamper-proof plaintext value is acceptable
// and mirrors the original Play session cookie (signed, not encrypted).
func newSecureCookie(secret string) *securecookie.SecureCookie {
	sc := securecookie.New([]byte(secret), nil)
	sc.MaxAge(sessionMaxAge)
	return sc
}

func AuthMiddleware(svc domain.AuthService, secret string) app.HandlerFunc {
	sc := newSecureCookie(secret)

	return func(ctx context.Context, c *app.RequestContext) {
		if !svc.IsEnabled() {
			c.Next(ctx)
			return
		}

		if username, ok := decodeSessionCookie(sc, c); ok {
			c.Set("user", &domain.User{Name: username})
			c.Next(ctx)
			return
		}

		if isAPICall(c) {
			c.JSON(http.StatusOK, domain.NullResponse())
			c.Abort()
			return
		}

		// Sanitize the stored target: the value is attacker-influenceable
		// (request URI) and later used in a Location header after login.
		redirectTo := SafeRedirectURL(string(c.Request.URI().RequestURI()))
		c.SetCookie(RedirectCookie, redirectTo, 300, "/", "", protocol.CookieSameSiteLaxMode, isTLSRequest(c), false)
		c.Redirect(http.StatusFound, []byte("/login"))
		c.Abort()
	}
}

func isAPICall(c *app.RequestContext) bool {
	accept := string(c.GetHeader("Accept"))
	if !strings.Contains(accept, "text/html") {
		return true
	}
	return strings.EqualFold(string(c.GetHeader("X-Requested-With")), "XMLHttpRequest")
}

// readCookie returns the named request cookie's value, undoing the
// url.QueryEscape that Hertz's SetCookie applies on write. Hertz's
// RequestContext.Cookie returns the raw header value (no decoding), so the
// escape must be reversed here or values never round-trip (e.g. the base64
// "=" padding in securecookie output becomes "%3D" and fails Decode).
func readCookie(c *app.RequestContext, name string) (string, bool) {
	raw := c.Cookie(name)
	if raw == nil {
		return "", false
	}
	v, err := url.QueryUnescape(string(raw))
	if err != nil {
		return "", false
	}
	return v, true
}

// decodeSessionCookie returns the username carried by a valid session cookie.
func decodeSessionCookie(sc *securecookie.SecureCookie, c *app.RequestContext) (string, bool) {
	value, ok := readCookie(c, SessionCookie)
	if !ok {
		return "", false
	}
	var username string
	if sc.Decode(SessionCookie, value, &username) != nil {
		return "", false
	}
	return username, true
}

// SessionUser reports the authenticated username carried by a valid session
// cookie, if any. Used by handlers that must stay reachable without the
// middleware (e.g. the login page) but still need the session state.
func SessionUser(c *app.RequestContext, secret string) (string, bool) {
	return decodeSessionCookie(newSecureCookie(secret), c)
}

// SetSessionCookie issues the signed session cookie for username. The Secure
// flag follows the request scheme so plain-HTTP deployments keep working
// (parity with the original); HttpOnly and SameSite=Lax are always set.
func SetSessionCookie(c *app.RequestContext, secret, username string) error {
	encoded, err := newSecureCookie(secret).Encode(SessionCookie, username)
	if err != nil {
		return err
	}
	c.SetCookie(SessionCookie, encoded, sessionMaxAge, "/", "", protocol.CookieSameSiteLaxMode, isTLSRequest(c), true)
	return nil
}

func ClearSessionCookie(c *app.RequestContext) {
	// Secure must match the flag used when the cookie was set, or the
	// browser will not expire it.
	c.SetCookie(SessionCookie, "", -1, "/", "", protocol.CookieSameSiteLaxMode, isTLSRequest(c), true)
}

// GetRedirectCookie returns the sanitized post-login redirect target.
func GetRedirectCookie(c *app.RequestContext) string {
	if value, ok := readCookie(c, RedirectCookie); ok {
		return SafeRedirectURL(value)
	}
	return "/"
}

// SafeRedirectURL returns s when it is a safe same-origin redirect target: a
// relative path starting with a single "/" — never "//" or "/\", which
// browsers treat as protocol-relative URLs (open redirect) — and free of
// control characters (which could split the Location header). Anything else
// yields "/".
func SafeRedirectURL(s string) string {
	if len(s) == 0 || s[0] != '/' || strings.HasPrefix(s, "//") || strings.HasPrefix(s, "/\\") {
		return "/"
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] == 0x7f {
			return "/"
		}
	}
	return s
}

// isTLSRequest reports whether the request arrived over TLS. Behind a
// plain-HTTP reverse proxy this is false (matching the original behavior).
func isTLSRequest(c *app.RequestContext) bool {
	return strings.EqualFold(string(c.Request.URI().Scheme()), "https")
}
