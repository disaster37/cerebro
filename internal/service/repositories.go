package service

import (
	"context"
	"encoding/json"

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

func (s *RepositoriesService) Verify(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	name, err := req.Get("name")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	forceFullSize, _ := req.GetBool("forceFullSize")

	verifyResp, err := s.snapshots.VerifyRepository(name, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	if !verifyResp.IsSuccess() {
		status := domain.RepositoryVerifyStatus{
			Name:     name,
			Verified: false,
			Status:   "ERROR",
			Nodes:    []domain.VerifiedRepoNode{},
			Error:    extractReason(verifyResp.Body),
		}
		body, _ := json.Marshal(status)
		return domain.NewResponse(200, body), nil
	}

	nodes, _ := domain.ParseRepositoryVerification(verifyResp.Body)

	snapshotsResp, err := s.snapshots.GetSnapshots(name, ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	if !snapshotsResp.IsSuccess() {
		return *snapshotsResp, nil
	}

	counts := snapshotCounts(snapshotsResp.Body)
	status := domain.RepositoryVerifyStatus{
		Name:          name,
		Verified:      true,
		Status:        "CONNECTED",
		NodeCount:     len(nodes),
		Nodes:         nodes,
		SnapshotCount: counts.total,
		ActiveCount:   counts.active,
		FailedCount:   counts.failed,
	}
	if counts.failed > 0 {
		status.Status = "PARTIAL_FAILURES"
	} else if counts.active > 0 {
		status.Status = "IN_PROGRESS"
	}

	size, err := domain.ParseRepositorySizeStats(snapshotsResp.Body)
	if err != nil {
		return domain.ErrorResponse(500, err.Error()), nil
	}
	if size.HasStats {
		status.SizeComputed = true
		status.SizeInBytes = &size.SizeInBytes
		status.LogicalSizeInBytes = &size.LogicalSizeInBytes
	} else if counts.total > 0 && (counts.total <= 50 || forceFullSize) {
		statusResp, serr := s.snapshots.GetSnapshotStatus(name, "_all", true, ctx, req.Target)
		if serr != nil {
			return domain.ErrorResponse(502, serr.Error()), nil
		}
		if !statusResp.IsSuccess() {
			return *statusResp, nil
		}
		size, serr = domain.ParseRepositorySizeStats(statusResp.Body)
		if serr != nil {
			return domain.ErrorResponse(500, serr.Error()), nil
		}
		status.SizeComputed = true
		status.SizeInBytes = &size.SizeInBytes
		status.LogicalSizeInBytes = &size.LogicalSizeInBytes
	} else if counts.total > 50 && !forceFullSize {
		status.SizeComputed = false
		status.SizeSkippedReason = "More than 50 snapshots; click Calculate Size to scan all shards"
	}

	body, _ := json.Marshal(status)
	return domain.NewResponse(200, body), nil
}

type snapshotCountSummary struct {
	total  int
	active int
	failed int
}

func snapshotCounts(in json.RawMessage) snapshotCountSummary {
	var raw struct {
		Snapshots []struct {
			State string `json:"state"`
		} `json:"snapshots"`
	}
	_ = json.Unmarshal(in, &raw)
	out := snapshotCountSummary{total: len(raw.Snapshots)}
	for _, snap := range raw.Snapshots {
		switch snap.State {
		case "IN_PROGRESS", "STARTED":
			out.active++
		case "FAILED", "PARTIAL", "INCOMPATIBLE":
			out.failed++
		}
	}
	return out
}

func extractReason(body json.RawMessage) string {
	var raw struct {
		Error json.RawMessage `json:"error"`
	}
	_ = json.Unmarshal(body, &raw)
	if len(raw.Error) > 0 {
		var s string
		if json.Unmarshal(raw.Error, &s) == nil {
			return s
		}
		var obj struct {
			Reason string `json:"reason"`
			Type   string `json:"type"`
		}
		if json.Unmarshal(raw.Error, &obj) == nil {
			if obj.Reason != "" {
				return obj.Reason
			}
			if obj.Type != "" {
				return obj.Type
			}
		}
	}
	return string(body)
}
