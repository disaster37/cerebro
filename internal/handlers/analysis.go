package handlers

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"cerebro/internal/models"
)

func (h *Handlers) AnalysisGetIndices(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		resp, err := h.Elastic.GetIndices(ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		body, err := models.OpenIndices(resp.Body)
		if err != nil {
			return models.ErrorResponse(500, err.Error()), nil
		}
		return models.NewResponse(resp.Status, body), nil
	})
}

func (h *Handlers) AnalysisGetIndexAnalyzers(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		index, err := req.Get("index")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.GetIndexSettings(index, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		body, err := models.IndexAnalyzers(index, resp.Body)
		if err != nil {
			return models.ErrorResponse(500, err.Error()), nil
		}
		return models.NewResponse(resp.Status, body), nil
	})
}

func (h *Handlers) AnalysisGetIndexFields(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		index, err := req.Get("index")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.GetIndexMapping(index, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		body, err := models.IndexFields(index, resp.Body)
		if err != nil {
			return models.ErrorResponse(500, err.Error()), nil
		}
		return models.NewResponse(resp.Status, body), nil
	})
}

func (h *Handlers) AnalysisAnalyzeByField(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		index, err := req.Get("index")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		field, err := req.Get("field")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		text, err := req.Get("text")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.AnalyzeTextByField(index, field, text, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		body, err := models.Tokens(resp.Body)
		if err != nil {
			return models.ErrorResponse(500, err.Error()), nil
		}
		return models.NewResponse(resp.Status, body), nil
	})
}

func (h *Handlers) AnalysisAnalyzeByAnalyzer(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		index, err := req.Get("index")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		analyzer, err := req.Get("analyzer")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		text, err := req.Get("text")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.AnalyzeTextByAnalyzer(index, analyzer, text, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		body, err := models.Tokens(resp.Body)
		if err != nil {
			return models.ErrorResponse(500, err.Error()), nil
		}
		return models.NewResponse(resp.Status, body), nil
	})
}
