package handlers

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/hertz/pkg/app"

	"cerebro/internal/models"
)

func (h *Handlers) OverviewIndex(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		apis := []string{
			"/_cluster/state/master_node,routing_table,blocks",
			"/_nodes/stats/jvm,fs,os,process?human=true",
			"/_stats/docs,store?ignore_unavailable=true",
			"/_cluster/settings",
			"/_aliases",
			"/_cluster/health",
			"/_nodes/_all/os,jvm?human=true",
		}
		responses := make([]json.RawMessage, len(apis))
		for i, api := range apis {
			resp, err := h.Elastic.ExecuteRequest("GET", api, nil, ctx, req.Target)
			if err != nil {
				return models.ErrorResponse(502, err.Error()), nil
			}
			if resp.Status < 200 || resp.Status >= 300 {
				body, _ := json.Marshal(map[string]string{"error": "ES request failed"})
				return models.Response{Status: resp.Status, Body: body}, nil
			}
			responses[i] = resp.Body
		}

		body, err := models.ClusterOverview(responses[0], responses[1], responses[2], responses[3], responses[4], responses[5], responses[6])
		if err != nil {
			return models.ErrorResponse(500, err.Error()), nil
		}
		return models.NewResponse(200, body), nil
	})
}

func (h *Handlers) OverviewDisableShardAllocation(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		kind, err := req.Get("kind")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.DisableShardAllocation(kind, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}

func (h *Handlers) OverviewEnableShardAllocation(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		resp, err := h.Elastic.EnableShardAllocation(ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}

func (h *Handlers) OverviewCloseIndices(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		idx, err := req.Get("indices")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.CloseIndex(idx, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}

func (h *Handlers) OverviewOpenIndices(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		idx, err := req.Get("indices")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.OpenIndex(idx, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}

func (h *Handlers) OverviewForceMerge(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		idx, err := req.Get("indices")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.ForceMerge(idx, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}

func (h *Handlers) OverviewClearIndexCache(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		idx, err := req.Get("indices")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.ClearIndexCache(idx, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}

func (h *Handlers) OverviewRefreshIndex(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		idx, err := req.Get("indices")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.RefreshIndex(idx, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}

func (h *Handlers) OverviewFlushIndex(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		idx, err := req.Get("indices")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.FlushIndex(idx, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}

func (h *Handlers) OverviewDeleteIndex(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		idx, err := req.Get("indices")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.DeleteIndex(idx, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}

func (h *Handlers) OverviewGetShardStats(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		index, err := req.Get("index")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		shard, err := req.GetInt("shard")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		node, err := req.Get("node")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		stats, err := h.Elastic.GetShardStats(index, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		recovery, err := h.Elastic.GetIndexRecovery(index, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		if stats.Status < 200 || stats.Status >= 300 {
			return *stats, nil
		}
		if recovery.Status < 200 || recovery.Status >= 300 {
			return *recovery, nil
		}
		body, err := models.ShardStats(index, node, shard, stats.Body, recovery.Body)
		if err != nil {
			return models.ErrorResponse(500, err.Error()), nil
		}
		return models.NewResponse(200, body), nil
	})
}

func (h *Handlers) OverviewRelocateShard(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		index, err := req.Get("index")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		shard, err := req.GetInt("shard")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		from, err := req.Get("from")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		to, err := req.Get("to")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.RelocateShard(shard, index, from, to, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}
