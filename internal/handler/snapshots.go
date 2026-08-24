package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handlers) SnapshotsGet(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Snapshots.Get)
}

func (h *Handlers) SnapshotsLoad(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Snapshots.Load)
}

func (h *Handlers) SnapshotsDelete(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Snapshots.Delete)
}

func (h *Handlers) SnapshotsCreate(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Snapshots.Create)
}

func (h *Handlers) SnapshotsRestore(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Snapshots.Restore)
}
