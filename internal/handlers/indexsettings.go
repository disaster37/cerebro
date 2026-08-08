package handlers

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"cerebro/internal/models"
)

func (h *Handlers) IndexSettingsGet(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		index, err := req.Get("index")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.GetIndexSettingsFlat(index, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}

func (h *Handlers) IndexSettingsUpdate(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		index, err := req.Get("index")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		settings, err := req.GetObj("settings")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.UpdateIndexSettings(index, settings, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}
