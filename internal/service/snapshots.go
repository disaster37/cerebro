package service

import (
	"context"
	"encoding/json"
	"strings"

	"cerebro/internal/domain"
)

type SnapshotsService struct {
	snapshots domain.SnapshotAPI
	indices   domain.IndexAPI
}

func NewSnapshotsService(snapshots domain.SnapshotAPI, indices domain.IndexAPI) *SnapshotsService {
	return &SnapshotsService{snapshots: snapshots, indices: indices}
}

func (s *SnapshotsService) Get(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	indicesResp, err := s.indices.GetIndices(ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	reposResp, err := s.snapshots.GetRepositories(ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	idx, _ := domain.SnapshotIndices(indicesResp.Body)
	repos, _ := domain.SnapshotRepoList(reposResp.Body)
	body, _ := json.Marshal(map[string]json.RawMessage{
		"indices":      idx,
		"repositories": repos,
	})
	return domain.NewResponse(200, body), nil
}

func (s *SnapshotsService) Load(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	repository, err := req.Get("repository")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.snapshots.GetSnapshots(repository, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	snapshots, err := domain.Snapshots(resp.Body)
	if err != nil {
		return domain.ErrorResponse(500, err.Error()), nil
	}
	return domain.NewResponse(resp.Status, snapshots), nil
}

func (s *SnapshotsService) Delete(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	repository, err := req.Get("repository")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	snapshot, err := req.Get("snapshot")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	resp, err := s.snapshots.DeleteSnapshot(repository, snapshot, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}

func (s *SnapshotsService) Create(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	repository, err := req.Get("repository")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	snapshot, err := req.Get("snapshot")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	indices := strings.Join(req.GetStringArray("indices"), ",")
	ignoreUnavailable, _ := req.GetBool("ignoreUnavailable")
	includeGlobalState, _ := req.GetBool("includeGlobalState")
	resp, err := s.snapshots.CreateSnapshot(repository, snapshot, ignoreUnavailable, includeGlobalState, indices, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}

func (s *SnapshotsService) Restore(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	repository, err := req.Get("repository")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	snapshot, err := req.Get("snapshot")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	renamePattern := req.GetOpt("renamePattern")
	renameReplacement := req.GetOpt("renameReplacement")
	indices := strings.Join(req.GetStringArray("indices"), ",")
	ignoreUnavailable, _ := req.GetBool("ignoreUnavailable")
	includeAliases, _ := req.GetBool("includeAliases")
	includeGlobalState, _ := req.GetBool("includeGlobalState")
	resp, err := s.snapshots.RestoreSnapshot(repository, snapshot, renamePattern, renameReplacement, ignoreUnavailable, includeAliases, includeGlobalState, indices, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	return *resp, nil
}
