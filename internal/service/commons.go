package service

import (
	"context"

	"cerebro/internal/domain"
)

type CommonsService struct {
	indices domain.IndexAPI
	nodes   domain.NodeAPI
}

func NewCommonsService(indices domain.IndexAPI, nodes domain.NodeAPI) *CommonsService {
	return &CommonsService{indices: indices, nodes: nodes}
}

func (s *CommonsService) Indices(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	resp, err := s.indices.GetIndices(ctx, req.Target)
	return transform(resp, err, domain.CommonsIndices)
}

func (s *CommonsService) Nodes(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	resp, err := s.nodes.GetNodes(ctx, req.Target)
	return transform(resp, err, domain.CommonsNodes)
}

func (s *CommonsService) GetIndexSettings(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	return forwardOp(ctx, req, "index", s.indices.GetIndexSettings)
}

func (s *CommonsService) GetIndexMappings(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	return forwardOp(ctx, req, "index", s.indices.GetIndexMapping)
}

func (s *CommonsService) GetNodeStats(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	return forwardOp(ctx, req, "node", s.nodes.NodeStats)
}

func (s *CommonsService) GetIndexStats(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	return forwardOp(ctx, req, "index", s.indices.IndexStats)
}
