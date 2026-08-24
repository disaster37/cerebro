package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handlers) AliasesGetAliases(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Aliases.GetAliases)
}

func (h *Handlers) AliasesUpdateAliases(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Aliases.UpdateAliases)
}
