package service

import (
	"context"
	"encoding/json"
	"strings"

	"cerebro/internal/domain"
)

type PoliciesService struct {
	cluster  domain.ClusterAPI
	policies domain.PolicyAPI
}

func NewPoliciesService(cluster domain.ClusterAPI, policies domain.PolicyAPI) *PoliciesService {
	return &PoliciesService{cluster: cluster, policies: policies}
}

func (s *PoliciesService) ListIndexPolicies(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	backendType := s.backendType(ctx, req)

	policiesResp, err := s.policies.GetIndexLifecyclePolicies(ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	if !policiesResp.IsSuccess() {
		if isUninitializedConfig(policiesResp) {
			return emptyIndexLifecycle(backendType), nil
		}
		if policiesResp.Status == 404 {
			return unsupportedIndexLifecycle(backendType), nil
		}
		return *policiesResp, nil
	}

	explainResp, err := s.policies.ExplainIndexLifecycle("*", ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	explainBody := json.RawMessage(`{}`)
	if explainResp.IsSuccess() {
		explainBody = explainResp.Body
	} else if explainResp.Status != 404 {
		return *explainResp, nil
	}

	statusResp, err := s.policies.GetIndexLifecycleStatus(ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	statusBody := json.RawMessage(`{}`)
	if statusResp.IsSuccess() {
		statusBody = statusResp.Body
	}

	body, err := domain.NormalizeIndexLifecycle(backendType, policiesResp.Body, explainBody, statusBody)
	if err != nil {
		return domain.ErrorResponse(500, err.Error()), nil
	}
	return domain.NewResponse(200, body), nil
}

func (s *PoliciesService) RetryIndices(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	indices := req.GetStringArray("indices")
	if len(indices) == 0 {
		return domain.ErrorResponse(400, "indices is required"), nil
	}
	return forward(s.policies.RetryIndexLifecycle(indices, req.GetOpt("state"), ctx, req.Target))
}

func (s *PoliciesService) ToggleEngine(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	start, err := req.GetBool("start")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	return forward(s.policies.SetIndexLifecycleRunning(start, ctx, req.Target))
}

func (s *PoliciesService) ListSnapshotPolicies(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	backendType := s.backendType(ctx, req)

	policiesResp, err := s.policies.GetSnapshotLifecyclePolicies(ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	if !policiesResp.IsSuccess() {
		if isUninitializedConfig(policiesResp) {
			return emptySnapshotLifecycle(), nil
		}
		if policiesResp.Status == 404 {
			return unsupportedSnapshotLifecycle(backendType), nil
		}
		return *policiesResp, nil
	}

	explainResp, err := s.policies.ExplainSnapshotLifecyclePolicies(ctx, req.Target)
	if err != nil {
		return domain.ErrorResponse(502, err.Error()), nil
	}
	explainBody := json.RawMessage(`{}`)
	if explainResp.IsSuccess() {
		explainBody = explainResp.Body
	} else if explainResp.Status != 404 {
		return *explainResp, nil
	}

	body, err := domain.NormalizeSnapshotLifecycle(backendType, policiesResp.Body, explainBody)
	if err != nil {
		return domain.ErrorResponse(500, err.Error()), nil
	}
	return domain.NewResponse(200, body), nil
}

func (s *PoliciesService) SnapshotAction(ctx context.Context, req domain.CerebroRequest) (domain.Response, error) {
	policyID, err := req.Get("policyId")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	action, err := req.Get("action")
	if err != nil {
		return domain.ErrorResponse(400, err.Error()), nil
	}
	return forward(s.policies.ExecuteOrToggleSnapshotPolicy(policyID, action, ctx, req.Target))
}

func (s *PoliciesService) backendType(ctx context.Context, req domain.CerebroRequest) string {
	bi, err := s.cluster.BackendInfo(ctx, req.Target)
	if err != nil {
		return "elasticsearch"
	}
	return bi.Type
}

func isUninitializedConfig(resp *domain.Response) bool {
	if resp.Status != 404 {
		return false
	}
	body := string(resp.Body)
	// OpenSearch lazily creates the ISM/SM config indices; before the first
	// policy is created, GET /_plugins/_ism/policies and /_plugins/_sm/policies
	// return 404 with "Configured indices are not found: [.opendistro-ism-config]"
	// (or "index_not_found_exception"). Treat that as an empty policy list, not
	// as a missing plugin.
	return strings.Contains(body, "index_not_found_exception") ||
		strings.Contains(body, "Configured indices are not found") ||
		strings.Contains(body, ".opendistro-ism-config") ||
		strings.Contains(body, ".opendistro-sm-config") ||
		strings.Contains(body, ".opensearch-sm-config")
}

func emptyIndexLifecycle(backendType string) domain.Response {
	feature := "ILM"
	if backendType == "opensearch" {
		feature = "ISM"
	}
	body, _ := json.Marshal(domain.IndexLifecyclePayload{
		BackendType:    backendType,
		FeatureName:    feature,
		OperationMode:  "RUNNING",
		Policies:       []domain.IndexPolicySummary{},
		ManagedIndices: []domain.ManagedIndexStatus{},
	})
	return domain.NewResponse(200, body)
}

func unsupportedIndexLifecycle(backendType string) domain.Response {
	feature := "ILM"
	if backendType == "opensearch" {
		feature = "ISM"
	}
	body, _ := json.Marshal(domain.IndexLifecyclePayload{
		BackendType:    backendType,
		FeatureName:    feature,
		OperationMode:  "UNKNOWN",
		Policies:       []domain.IndexPolicySummary{},
		ManagedIndices: []domain.ManagedIndexStatus{},
		Unsupported:    true,
		Warning:        feature + " is not available on this cluster (plugin disabled or unsupported distribution)",
	})
	return domain.NewResponse(200, body)
}

func unsupportedSnapshotLifecycle(backendType string) domain.Response {
	feature := "SLM"
	if backendType == "opensearch" {
		feature = "SM"
	}
	body, _ := json.Marshal(map[string]any{
		"backendType": backendType,
		"featureName": feature,
		"policies":    []domain.SnapshotPolicySummary{},
		"unsupported": true,
		"warning":     feature + " is not available on this cluster (plugin disabled or unsupported distribution)",
	})
	return domain.NewResponse(200, body)
}

// emptySnapshotLifecycle returns the same shape as a successful (but empty)
// snapshot lifecycle list: a JSON array of zero policies.
func emptySnapshotLifecycle() domain.Response {
	return domain.NewResponse(200, json.RawMessage("[]"))
}
