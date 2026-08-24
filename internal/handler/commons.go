package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handlers) CommonsIndices(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Commons.Indices)
}

func (h *Handlers) CommonsNodes(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Commons.Nodes)
}

func (h *Handlers) CommonsGetIndexSettings(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Commons.GetIndexSettings)
}

func (h *Handlers) CommonsGetIndexMappings(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Commons.GetIndexMappings)
}

func (h *Handlers) CommonsGetNodeStats(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Commons.GetNodeStats)
}

func (h *Handlers) CommonsGetIndexStats(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Commons.GetIndexStats)
}
