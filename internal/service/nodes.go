package service

import (
	"context"
	"encoding/json"

	"cerebro/internal/domain"
)

type NodesService struct {
	nodes domain.NodeAPI
	cat   domain.CatAPI
}

func NewNodesService(nodes domain.NodeAPI, cat domain.CatAPI) *NodesService {
	return &NodesService{nodes: nodes, cat: cat}
}

func (s *NodesService) Index(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	info, err := s.nodes.Nodes([]string{"jvm", "os"}, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	if !info.IsSuccess() {
		return *info, nil
	}
	stats, err := s.nodes.NodesStats([]string{"jvm", "fs", "os", "process"}, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	if !stats.IsSuccess() {
		return *stats, nil
	}
	// OpenSearch's cgroup-aware resource tracker is a best-effort secondary
	// source; Elasticsearch or older/restricted clusters simply fall back.
	var resourceStatsRaw json.RawMessage
	if s.nodes != nil {
		if res, rerr := s.nodes.NodesResourceUsageStats(ctx, req.Target); rerr == nil && res != nil && res.IsSuccess() {
			resourceStatsRaw = res.Body
		}
	}
	master, err := s.cat.CatMaster(ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	if !master.IsSuccess() {
		return *master, nil
	}
	body, err := domain.NodesData(info.Body, stats.Body, master.Body, resourceStatsRaw)
	if err != nil {
		return domain.ErrorResponse(500, err.Error()), nil
	}
	return domain.NewResponse(200, body), nil
}
