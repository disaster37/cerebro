package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handlers) NavbarIndex(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Navbar.Index)
}
