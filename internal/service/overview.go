package service

import (
	"context"
	"encoding/json"

	"cerebro/internal/domain"
)

type OverviewService struct {
	raw     domain.RawAPI
	cluster domain.ClusterAPI
	indices domain.IndexAPI
}

func NewOverviewService(raw domain.RawAPI, cluster domain.ClusterAPI, indices domain.IndexAPI) *OverviewService {
	return &OverviewService{raw: raw, cluster: cluster, indices: indices}
}

func (s *OverviewService) Index(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
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
		resp, err := s.raw.ExecuteRequest("GET", api, nil, ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		if resp.Status < 200 || resp.Status >= 300 {
			body, _ := json.Marshal(map[string]string{"error": "ES request failed"})
			return domain.Response{Status: resp.Status, Body: body}, nil
		}
		responses[i] = resp.Body
	}

	body, err := domain.ClusterOverview(responses[0], responses[1], responses[2], responses[3], responses[4], responses[5], responses[6])
	if err != nil {
		return domain.ErrorResponse(500, err.Error()), nil
	}
	return domain.NewResponse(200, body), nil
}

func (s *OverviewService) DisableShardAllocation(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	kind, err := req.Get("kind")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.cluster.DisableShardAllocation(kind, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}

func (s *OverviewService) EnableShardAllocation(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	resp, err := s.cluster.EnableShardAllocation(ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}

func (s *OverviewService) CloseIndices(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	idx, err := req.Get("indices")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.indices.CloseIndex(idx, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}

func (s *OverviewService) OpenIndices(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	idx, err := req.Get("indices")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.indices.OpenIndex(idx, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}

func (s *OverviewService) ForceMerge(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	idx, err := req.Get("indices")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.indices.ForceMerge(idx, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}

func (s *OverviewService) ClearIndexCache(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	idx, err := req.Get("indices")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.indices.ClearIndexCache(idx, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}

func (s *OverviewService) RefreshIndex(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	idx, err := req.Get("indices")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.indices.RefreshIndex(idx, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}

func (s *OverviewService) FlushIndex(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	idx, err := req.Get("indices")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.indices.FlushIndex(idx, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}

func (s *OverviewService) DeleteIndex(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	idx, err := req.Get("indices")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.indices.DeleteIndex(idx, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}

func (s *OverviewService) GetShardStats(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	index, err := req.Get("index")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	shard, err := req.GetInt("shard")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	node, err := req.Get("node")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	stats, err := s.indices.GetShardStats(index, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	recovery, err := s.indices.GetIndexRecovery(index, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	if stats.Status < 200 || stats.Status >= 300 {
		return *stats, nil
	}
	if recovery.Status < 200 || recovery.Status >= 300 {
		return *recovery, nil
	}
	body, err := domain.ShardStats(index, node, shard, stats.Body, recovery.Body)
	if err != nil {
		return domain.ErrorResponse(500, err.Error()), nil
	}
	return domain.NewResponse(200, body), nil
}

func (s *OverviewService) RelocateShard(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	index, err := req.Get("index")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	shard, err := req.GetInt("shard")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	from, err := req.Get("from")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	to, err := req.Get("to")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.cluster.RelocateShard(shard, index, from, to, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}
