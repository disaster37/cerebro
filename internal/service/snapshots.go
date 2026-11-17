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
	if !reposResp.IsSuccess() {
		return *reposResp, nil
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
	return transform(resp, err, domain.Snapshots)
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
	return forward(s.snapshots.DeleteSnapshot(repository, snapshot, ctx, req.Target))
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
	return forward(s.snapshots.CreateSnapshot(repository, snapshot, ignoreUnavailable, includeGlobalState, indices, ctx, req.Target))
}

func boolOpt(req domain.CerebroRequest, name string) bool {
	v, _ := req.GetBool(name)
	return v
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
	opts := domain.RestoreSnapshotOptions{
		Indices:             strings.Join(req.GetStringArray("indices"), ","),
		RenamePattern:       req.GetOpt("renamePattern"),
		RenameReplacement:   req.GetOpt("renameReplacement"),
		IgnoreUnavailable:   boolOpt(req, "ignoreUnavailable"),
		IncludeAliases:      boolOpt(req, "includeAliases"),
		IncludeGlobalState:  boolOpt(req, "includeGlobalState"),
		Partial:             boolOpt(req, "partial"),
		IndexSettings:       req.GetObjOpt("indexSettings"),
		IgnoreIndexSettings: req.GetStringArray("ignoreIndexSettings"),
	}
	return forward(s.snapshots.RestoreSnapshot(repository, snapshot, opts, ctx, req.Target))
}
