package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handlers) RestIndex(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Rest.Index)
}

func (h *Handlers) RestRequest(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Rest.Request)
}

func (h *Handlers) RestHistory(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Rest.History)
}
