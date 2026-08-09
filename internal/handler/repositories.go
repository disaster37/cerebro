package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handlers) RepositoriesGet(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Repositories.Get)
}

func (h *Handlers) RepositoriesSave(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Repositories.Save)
}

func (h *Handlers) RepositoriesDelete(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Repositories.Delete)
}
