package handlers

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"

	"cerebro/internal/auth"
	"cerebro/internal/models"
)

func (h *Handlers) AuthLogin(ctx context.Context, c *app.RequestContext) {
	user := string(c.FormValue("user"))
	password := string(c.FormValue("password"))

	if user == "" || password == "" {
		c.JSON(http.StatusOK, models.ErrorResponse(400, "invalid login form data"))
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
