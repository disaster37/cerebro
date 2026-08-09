package handlers

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"cerebro/internal/domain"
)

func (h *Handlers) NodesIndex(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
		info, err := h.Elastic.Nodes([]string{"jvm", "os"}, ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		if info.Status < 200 || info.Status >= 300 {
			return *info, nil
		}
		stats, err := h.Elastic.NodesStats([]string{"jvm", "fs", "os", "process"}, ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		if stats.Status < 200 || stats.Status >= 300 {
			return *stats, nil
		}
		master, err := h.Elastic.CatMaster(ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		if master.Status < 200 || master.Status >= 300 {
			return *master, nil
		}
		body, err := domain.NodesData(info.Body, stats.Body, master.Body)
		if err != nil {
			return domain.ErrorResponse(500, err.Error()), nil
		}
		return domain.NewResponse(200, body), nil
	})
}
