package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handlers) CreateIndexGetIndexMetadata(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.CreateIndex.GetIndexMetadata)
}

func (h *Handlers) CreateIndexExecute(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.CreateIndex.Execute)
}
