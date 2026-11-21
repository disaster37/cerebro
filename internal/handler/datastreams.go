package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handlers) DataStreamsList(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.DataStreams.List)
}

func (h *Handlers) DataStreamsCreate(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.DataStreams.Create)
}

func (h *Handlers) DataStreamsDelete(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.DataStreams.Delete)
}

func (h *Handlers) DataStreamsRollover(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.DataStreams.Rollover)
}
