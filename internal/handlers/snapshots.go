package handlers

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	"cerebro/internal/models"
)

func (h *Handlers) SnapshotsGet(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		indicesResp, err := h.Elastic.GetIndices(ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		reposResp, err := h.Elastic.GetRepositories(ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		idx, _ := models.SnapshotIndices(indicesResp.Body)
		repos, _ := models.SnapshotRepoList(reposResp.Body)
		body, _ := json.Marshal(map[string]json.RawMessage{
			"indices":      idx,
			"repositories": repos,
		})
		return models.NewResponse(200, body), nil
	})
}

func (h *Handlers) SnapshotsLoad(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		repository, err := req.Get("repository")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.GetSnapshots(repository, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		snapshots, err := models.Snapshots(resp.Body)
		if err != nil {
			return models.ErrorResponse(500, err.Error()), nil
		}
		return models.NewResponse(resp.Status, snapshots), nil
	})
}

func (h *Handlers) SnapshotsDelete(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		repository, err := req.Get("repository")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		snapshot, err := req.Get("snapshot")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		resp, err := h.Elastic.DeleteSnapshot(repository, snapshot, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}

func (h *Handlers) SnapshotsCreate(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		repository, err := req.Get("repository")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		snapshot, err := req.Get("snapshot")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		indices := strings.Join(req.GetStringArray("indices"), ",")
		ignoreUnavailable, _ := req.GetBool("ignoreUnavailable")
		includeGlobalState, _ := req.GetBool("includeGlobalState")
		resp, err := h.Elastic.CreateSnapshot(repository, snapshot, ignoreUnavailable, includeGlobalState, indices, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}

func (h *Handlers) SnapshotsRestore(ctx context.Context, c *app.RequestContext) {
	h.process(ctx, c, func(ctx context.Context, req CerebroRequest) (models.Response, error) {
		repository, err := req.Get("repository")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		snapshot, err := req.Get("snapshot")
		if err != nil {
			return models.ErrorResponse(400, err.Error()), nil
		}
		renamePattern := req.GetOpt("renamePattern")
		renameReplacement := req.GetOpt("renameReplacement")
		indices := strings.Join(req.GetStringArray("indices"), ",")
		ignoreUnavailable, _ := req.GetBool("ignoreUnavailable")
		includeAliases, _ := req.GetBool("includeAliases")
		includeGlobalState, _ := req.GetBool("includeGlobalState")
		resp, err := h.Elastic.RestoreSnapshot(repository, snapshot, renamePattern, renameReplacement, ignoreUnavailable, includeAliases, includeGlobalState, indices, ctx, req.Target)
		if err != nil {
			return models.ErrorResponse(502, err.Error()), nil
		}
		return *resp, nil
	})
}
