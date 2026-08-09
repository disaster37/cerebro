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
	api, err := req.Get("api")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.cat.CatRequest(api, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}
