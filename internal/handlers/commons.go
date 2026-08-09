package handlers

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"cerebro/internal/domain"
)

func (h *Handlers) CommonsIndices(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
		resp, err := h.Elastic.GetIndices(ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		body, err := domain.CommonsIndices(resp.Body)
		if err != nil {
			return domain.ErrorResponse(500, err.Error()), nil
		}
		return domain.NewResponse(resp.Status, body), nil
	})
}

func (h *Handlers) CommonsNodes(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
		resp, err := h.Elastic.GetNodes(ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		body, err := domain.CommonsNodes(resp.Body)
		if err != nil {
			return domain.ErrorResponse(500, err.Error()), nil
		}
		return domain.NewResponse(resp.Status, body), nil
	})
}

func (h *Handlers) CommonsGetIndexSettings(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
		index, err := req.Get("index")
		if err != nil {
			return domain.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.GetIndexSettings(index, ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}

func (h *Handlers) CommonsGetIndexMappings(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
		index, err := req.Get("index")
		if err != nil {
			return domain.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.GetIndexMapping(index, ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}

func (h *Handlers) CommonsGetNodeStats(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
		node, err := req.Get("node")
		if err != nil {
			return domain.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.NodeStats(node, ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}

func (h *Handlers) CommonsGetIndexStats(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
		index, err := req.Get("index")
		if err != nil {
			return domain.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.IndexStats(index, ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}
