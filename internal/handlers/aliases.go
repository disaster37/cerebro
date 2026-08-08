package handlers

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/hertz/pkg/app"

	"cerebro/internal/models"
)

func (h *Handlers) AliasesGetAliases(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		resp, err := h.Elastic.GetAliases(ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		aliases, err := models.Aliases(resp.Body)
		if err != nil {
			return models.ErrorResponse(500, err.Error()), nil
		}
		return models.NewResponse(resp.Status, aliases), nil
	})
}

func (h *Handlers) AliasesUpdateAliases(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		var m map[string]json.RawMessage
		_ = json.Unmarshal(req.Body, &m)
		changes := json.RawMessage("[]")
		if cg, ok := m["changes"]; ok {
			changes = cg
		}
		resp, err := h.Elastic.UpdateAliases(changes, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}
