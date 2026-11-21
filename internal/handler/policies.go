package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handlers) PoliciesIndexLifecycle(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Policies.ListIndexPolicies)
}

func (h *Handlers) PoliciesRetryIndices(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Policies.RetryIndices)
}

func (h *Handlers) PoliciesToggleEngine(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Policies.ToggleEngine)
}

func (h *Handlers) PoliciesSnapshotLifecycle(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Policies.ListSnapshotPolicies)
}

func (h *Handlers) PoliciesSnapshotAction(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.Policies.SnapshotAction)
}
