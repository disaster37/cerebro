package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handlers) OverviewIndex(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Overview.Index)
}

func (h *Handlers) OverviewDisableShardAllocation(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Overview.DisableShardAllocation)
}

func (h *Handlers) OverviewEnableShardAllocation(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Overview.EnableShardAllocation)
}

func (h *Handlers) OverviewCloseIndices(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Overview.CloseIndices)
}

func (h *Handlers) OverviewOpenIndices(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Overview.OpenIndices)
}

func (h *Handlers) OverviewForceMerge(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Overview.ForceMerge)
}

func (h *Handlers) OverviewClearIndexCache(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Overview.ClearIndexCache)
}

func (h *Handlers) OverviewRefreshIndex(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Overview.RefreshIndex)
}

func (h *Handlers) OverviewFlushIndex(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Overview.FlushIndex)
}

func (h *Handlers) OverviewDeleteIndex(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Overview.DeleteIndex)
}

func (h *Handlers) OverviewGetShardStats(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Overview.GetShardStats)
}

func (h *Handlers) OverviewRelocateShard(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Overview.RelocateShard)
}
