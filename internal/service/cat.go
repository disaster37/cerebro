package service

import (
	"context"

	"cerebro/internal/domain"
)

type CatService struct {
	cat domain.CatAPI
}

func NewCatService(cat domain.CatAPI) *CatService {
	return &CatService{cat: cat}
}

func (s *CatService) Get(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	return forwardOp(ctx, req, "api", s.cat.CatRequest)
}
