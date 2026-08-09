package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handlers) NodesIndex(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Nodes.Index)
}
