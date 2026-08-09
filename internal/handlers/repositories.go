package handlers

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"cerebro/internal/domain"
)

func (h *Handlers) RepositoriesGet(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
		resp, err := h.Elastic.GetRepositories(ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		body, err := domain.Repositories(resp.Body)
		if err != nil {
			return domain.ErrorResponse(500, err.Error()), nil
		}
		return domain.NewResponse(resp.Status, body), nil
	})
}

func (h *Handlers) RepositoriesSave(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
		name, err := req.Get("name")
		if err != nil {
			return domain.ErrorResponse(400, err.Error()), nil
		}
		repoType, err := req.Get("type")
		if err != nil {
			return domain.ErrorResponse(400, err.Error()), nil
		}
		settings, err := req.GetObj("settings")
		if err != nil {
			return domain.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.CreateRepository(name, repoType, settings, ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}

func (h *Handlers) RepositoriesDelete(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
		name, err := req.Get("name")
		if err != nil {
			return domain.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.DeleteRepository(name, ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}
