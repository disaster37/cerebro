package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handlers) ClusterSettingsGet(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.ClusterSettings.Get)
}

func (h *Handlers) ClusterSettingsSave(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, h.ClusterSettings.Save)
}
