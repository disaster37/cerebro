package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"

	"cerebro/internal/domain"
)

func (h *Handlers) ConnectIndex(ctx context.Context, c *app.RequestContext) {
	names := h.Hosts.GetHostNames()
	if names == nil {
		names = []string{}
	}
	body, _ := json.Marshal(names)
	c.JSON(http.StatusOK, domain.NewResponse(200, body))
}

func (h *Handlers) ConnectConnect(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
		resp, err := h.Elastic.ExecuteRequest("GET", "/_cluster/health", nil, ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}
