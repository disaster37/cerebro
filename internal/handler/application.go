package handler

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"

	"cerebro/internal/auth"
)

func (h *Handlers) ApplicationIndex(ctx context.Context, c *app.RequestContext) {
	if h.Auth.IsEnabled() {
		if _, ok := c.Get("user"); !ok {
			c.Redirect(http.StatusFound, []byte("/login"))
			return
		}
	}
	c.File("output/public/index.html")
}

func (h *Handlers) AuthIndex(ctx context.Context, c *app.RequestContext) {
	if !h.Auth.IsEnabled() {
		c.Redirect(http.StatusFound, []byte("/"))
		return
	}
	// /login is intentionally NOT behind AuthMiddleware (it must render for
	// unauthenticated users), so inspect the session cookie directly to keep
	// the "already logged in -> back to the app" behavior.
	if _, ok := auth.SessionUser(c, h.Secret); ok {
		c.Redirect(http.StatusFound, []byte("/"))
		return
	}
	c.File("output/public/index.html")
}
