package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handlers) IndexSettingsGet(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.IndexSettings.Get)
}

func (h *Handlers) IndexSettingsUpdate(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.IndexSettings.Update)
}
