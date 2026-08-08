package handlers

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"cerebro/internal/models"
)

func (h *Handlers) TemplatesTemplates(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		resp, err := h.Elastic.GetTemplates(ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		body, err := models.Templates(resp.Body)
		if err != nil {
			return models.ErrorResponse(500, err.Error()), nil
		}
		return models.NewResponse(resp.Status, body), nil
	})
}

func (h *Handlers) TemplatesDelete(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		name, err := req.Get("name")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.DeleteTemplate(name, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}

func (h *Handlers) TemplatesCreate(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		name, err := req.Get("name")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		template, err := req.GetObj("template")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.CreateTemplate(name, template, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}
