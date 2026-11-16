package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"

	"cerebro/internal/auth"
	"cerebro/internal/domain"
)

func (h *Handlers) AuthLogin(ctx context.Context, c *app.RequestContext) {
	user := string(c.FormValue("user"))
	password := string(c.FormValue("password"))

	if user == "" || password == "" {
		c.JSON(http.StatusOK, domain.ErrorResponse(400, "invalid login form data"))
		return
	}

	if !h.Auth.IsEnabled() {
		c.Redirect(http.StatusFound, []byte("/"))
		return
	}

	username, ok := h.Auth.Auth(user, password)
	if !ok {
		c.Redirect(http.StatusFound, []byte("/login"))
		return
	}

	if err := auth.SetSessionCookie(c, h.Secret, username); err != nil {
		// Misconfigured signing secret: fail closed instead of issuing an
		// empty/unsigned session.
		c.Redirect(http.StatusFound, []byte("/login"))
		return
	}
	// GetRedirectCookie sanitizes the target (same-origin relative path only),
	// preventing open redirects from an attacker-influenceable cookie value.
	redirectTo := auth.GetRedirectCookie(c)
	c.Redirect(http.StatusFound, []byte(redirectTo))
}

func (h *Handlers) AuthLogout(ctx context.Context, c *app.RequestContext) {
	auth.ClearSessionCookie(c)
	c.Redirect(http.StatusFound, []byte("/login"))
}

// AuthMeResponse is the JSON body returned inside the {status, body} envelope
// of GET /auth/me so the SPA can determine whether Cerebro authentication is
// enabled and restore the signed session username across browser reloads.
type AuthMeResponse struct {
	AuthEnabled bool   `json:"authEnabled"`
	Username    string `json:"username,omitempty"`
}

func (h *Handlers) AuthMe(ctx context.Context, c *app.RequestContext) {
	enabled := h.Auth != nil && h.Auth.IsEnabled()
	resp := AuthMeResponse{AuthEnabled: enabled}
	if enabled {
		if username, ok := auth.SessionUser(c, h.Secret); ok {
			resp.Username = username
		}
	}
	body, err := json.Marshal(resp)
	if err != nil {
		c.JSON(http.StatusOK, domain.ErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, domain.NewResponse(200, body))
}
