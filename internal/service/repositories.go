package service

import (
	"context"
	"encoding/json"
	"strings"

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

	switch {
	case counts.total == 0:
		zero := int64(0)
		status.SizeComputed = true
		status.SizeInBytes = &zero
		status.LogicalSizeInBytes = &zero
	case counts.total <= 50 || forceFullSize:
		// GET /_snapshot/{repo}/{snapshot}/_status is the authoritative
		// file-size endpoint on both Elasticsearch 8/9 and OpenSearch 2/3.
		// Fall back to the inline SnapshotInfo stats only when _status is
		// unavailable.
		size, serr := s.snapshotSizeFromStatus(name, snapshotNames(snapshotsResp.Body), ctx, req.Target)
		if serr != nil {
			return domain.ErrorResponse(502, serr.Error()), nil
		}
		if size == nil {
			inline, perr := domain.ParseRepositorySizeStats(snapshotsResp.Body)
			if perr != nil {
				return domain.ErrorResponse(500, perr.Error()), nil
			}
			size = &inline
		}
		status.SizeComputed = true
		status.SizeInBytes = &size.SizeInBytes
		status.LogicalSizeInBytes = &size.LogicalSizeInBytes
	default:
		size, perr := domain.ParseRepositorySizeStats(snapshotsResp.Body)
		if perr != nil {
			return domain.ErrorResponse(500, perr.Error()), nil
		}
		if size.HasStats && (size.SizeInBytes > 0 || size.LogicalSizeInBytes > 0) {
			status.SizeComputed = true
			status.SizeInBytes = &size.SizeInBytes
			status.LogicalSizeInBytes = &size.LogicalSizeInBytes
		} else {
			status.SizeComputed = false
			status.SizeSkippedReason = "More than 50 snapshots; click Calculate Size to scan all shards"
		}
	}

	body, _ := json.Marshal(status)
	return domain.NewResponse(200, body), nil
}

// snapshotSizeFromStatus queries the authoritative _status endpoint. It first
// tries the documented `_all` wildcard, then the explicit snapshot names:
// Elasticsearch 9 and OpenSearch 3 return an empty snapshot list for
// `_all/_status` even though the named form returns full stats. It returns
// (nil, nil) when no usable stats are available so the caller can fall back to
// the inline SnapshotInfo stats.
func (s *RepositoriesService) snapshotSizeFromStatus(name string, snapshots []string, ctx context.Context, target domain.ElasticServer) (*domain.RepositorySizeStats, error) {
	candidates := []string{"_all"}
	if len(snapshots) > 0 {
		candidates = append(candidates, strings.Join(snapshots, ","))
	}
	for _, snapshot := range candidates {
		statusResp, err := s.snapshots.GetSnapshotStatus(name, snapshot, true, ctx, target)
		if err != nil {
			return nil, err
		}
		if statusResp == nil || !statusResp.IsSuccess() {
			continue
		}
		size, err := domain.ParseRepositorySizeStats(statusResp.Body)
		if err != nil {
			return nil, err
		}
		if size.HasStats && (size.SizeInBytes > 0 || size.LogicalSizeInBytes > 0) {
			return &size, nil
		}
	}
	return nil, nil
}

func snapshotNames(in json.RawMessage) []string {
	var raw struct {
		Snapshots []struct {
			Snapshot string `json:"snapshot"`
		} `json:"snapshots"`
	}
	_ = json.Unmarshal(in, &raw)
	names := make([]string, 0, len(raw.Snapshots))
	for _, snap := range raw.Snapshots {
		if snap.Snapshot != "" {
			names = append(names, snap.Snapshot)
		}
	}
	return names
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
