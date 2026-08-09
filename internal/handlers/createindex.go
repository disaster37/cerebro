package handlers

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/hertz/pkg/app"

	"cerebro/internal/domain"
)

func (h *Handlers) CreateIndexGetIndexMetadata(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
		index, err := req.Get("index")
		if err != nil {
			return domain.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.GetIndexMetadata(index, ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		body, err := domain.IndexMetadata(resp.Body)
		if err != nil {
			return domain.ErrorResponse(500, err.Error()), nil
		}
		return domain.NewResponse(resp.Status, body), nil
	})
}

func (h *Handlers) CreateIndexExecute(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
		index, err := req.Get("index")
		if err != nil {
			return domain.ErrorResponse(400, err.Error()), nil
		}
		metadata := req.GetObjOpt("metadata")
		if metadata == nil {
			metadata = json.RawMessage("{}")
		}
		resp, err := h.Elastic.CreateIndex(index, metadata, ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}
