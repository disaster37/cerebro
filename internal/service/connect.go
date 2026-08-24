package service

import (
	"context"

	"cerebro/internal/domain"
)

type ConnectService struct {
	raw domain.RawAPI
}

func NewConnectService(raw domain.RawAPI) *ConnectService {
	return &ConnectService{raw: raw}
}

func (s *ConnectService) Connect(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	return forward(s.raw.ExecuteRequest("GET", "/_cluster/health", nil, ctx, req.Target))
}
