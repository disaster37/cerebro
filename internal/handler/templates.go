package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handlers) TemplatesTemplates(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Templates.Templates)
}

func (h *Handlers) TemplatesDelete(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Templates.Delete)
}

func (h *Handlers) TemplatesCreate(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Templates.Create)
}
