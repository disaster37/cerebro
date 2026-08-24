package service

import (
	"context"

	"cerebro/internal/domain"
)

type RepositoriesService struct {
	snapshots domain.SnapshotAPI
}

func NewRepositoriesService(snapshots domain.SnapshotAPI) *RepositoriesService {
	return &RepositoriesService{snapshots: snapshots}
}

func (s *RepositoriesService) Get(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	resp, err := s.snapshots.GetRepositories(ctx, req.Target)
	return transform(resp, err, domain.Repositories)
}

func (s *RepositoriesService) Save(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	name, err := req.Get("name")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	repoType, err := req.Get("type")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	settings, err := req.GetObj("settings")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	return forward(s.snapshots.CreateRepository(name, repoType, settings, ctx, req.Target))
}

func (s *RepositoriesService) Delete(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	return forwardOp(ctx, req, "name", s.snapshots.DeleteRepository)
}
