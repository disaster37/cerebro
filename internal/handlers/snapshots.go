package handlers

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	"cerebro/internal/domain"
)

func (h *Handlers) SnapshotsGet(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
		indicesResp, err := h.Elastic.GetIndices(ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		reposResp, err := h.Elastic.GetRepositories(ctx, req.Target)
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
	})
}

func (h *Handlers) SnapshotsLoad(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
		repository, err := req.Get("repository")
		if err != nil {
			return domain.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.GetSnapshots(repository, ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		snapshots, err := domain.Snapshots(resp.Body)
		if err != nil {
			return domain.ErrorResponse(500, err.Error()), nil
		}
		return domain.NewResponse(resp.Status, snapshots), nil
	})
}

func (h *Handlers) SnapshotsDelete(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
		repository, err := req.Get("repository")
		if err != nil {
			return domain.ErrorResponse(400, err.Error()), nil
		}
		snapshot, err := req.Get("snapshot")
		if err != nil {
			return domain.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.DeleteSnapshot(repository, snapshot, ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}

func (h *Handlers) SnapshotsCreate(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
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
		resp, err := h.Elastic.CreateSnapshot(repository, snapshot, ignoreUnavailable, includeGlobalState, indices, ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}

func (h *Handlers) SnapshotsRestore(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
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
		resp, err := h.Elastic.RestoreSnapshot(repository, snapshot, renamePattern, renameReplacement, ignoreUnavailable, includeAliases, includeGlobalState, indices, ctx, req.Target)
		if err != nil {
			return domain.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}
