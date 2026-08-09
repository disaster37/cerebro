package handlers

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"cerebro/internal/domain"
)

func (h *Handlers) CatGet(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
		api, err := req.Get("api")
		if err != nil {
			return domain.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.CatRequest(api, ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}
