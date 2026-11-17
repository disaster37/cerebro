package service

import (
	"context"
	"encoding/json"
	"testing"

	"cerebro/internal/domain"
)

type fakePolicyAPI struct {
	domain.PolicyAPI
	indexPoliciesResp *domain.Response
	explainResp       *domain.Response
	statusResp        *domain.Response
	snapshotResp      *domain.Response
	snapshotExplain   *domain.Response
}

func (f *fakePolicyAPI) GetIndexLifecyclePolicies(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return f.indexPoliciesResp, nil
}

func (f *fakePolicyAPI) ExplainIndexLifecycle(indexPattern string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return f.explainResp, nil
}

func (f *fakePolicyAPI) GetIndexLifecycleStatus(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return f.statusResp, nil
}

func (f *fakePolicyAPI) GetSnapshotLifecyclePolicies(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return f.snapshotResp, nil
}

func (f *fakePolicyAPI) ExplainSnapshotLifecyclePolicies(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return f.snapshotExplain, nil
}

type fakeClusterAPI struct {
	domain.ClusterAPI
	backendType string
}

func (f *fakeClusterAPI) BackendInfo(ctx context.Context, target domain.ElasticServer) (domain.BackendInfo, error) {
	return domain.BackendInfo{Type: f.backendType}, nil
}

func policyRequest() domain.CerebroRequest {
	return domain.CerebroRequest{Target: domain.ElasticServer{Host: domain.Host{Name: "http://os:9200"}}}
}

func TestListIndexPoliciesUninitializedISM(t *testing.T) {
	fake := &fakePolicyAPI{
		indexPoliciesResp: &domain.Response{Status: 404, Body: json.RawMessage(`{"error":{"type":"index_not_found_exception","reason":"no such index [.opendistro-ism-config]"}}`)},
	}
	svc := NewPoliciesService(&fakeClusterAPI{backendType: "opensearch"}, fake)
	resp, err := svc.ListIndexPolicies(context.Background(), policyRequest())
	if err != nil {
		t.Fatalf("ListIndexPolicies: %v", err)
	}
	if resp.Status != 200 {
		t.Fatalf("status = %d, want 200", resp.Status)
	}
	var out domain.IndexLifecyclePayload
	if err := json.Unmarshal(resp.Body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Unsupported || out.FeatureName != "ISM" || len(out.Policies) != 0 {
		t.Errorf("payload = %+v, want empty ISM list", out)
	}
}

func TestListIndexPoliciesUnsupportedPlugin(t *testing.T) {
	fake := &fakePolicyAPI{
		indexPoliciesResp: &domain.Response{Status: 404, Body: json.RawMessage(`{"error":"no handler found"}`)},
	}
	svc := NewPoliciesService(&fakeClusterAPI{backendType: "elasticsearch"}, fake)
	resp, err := svc.ListIndexPolicies(context.Background(), policyRequest())
	if err != nil {
		t.Fatalf("ListIndexPolicies: %v", err)
	}
	var out domain.IndexLifecyclePayload
	_ = json.Unmarshal(resp.Body, &out)
	if !out.Unsupported || out.Warning == "" {
		t.Errorf("payload = %+v, want unsupported warning", out)
	}
}

func TestListSnapshotPoliciesUnsupported(t *testing.T) {
	fake := &fakePolicyAPI{
		snapshotResp: &domain.Response{Status: 404, Body: json.RawMessage(`{"error":"no handler found"}`)},
	}
	svc := NewPoliciesService(&fakeClusterAPI{backendType: "opensearch"}, fake)
	resp, err := svc.ListSnapshotPolicies(context.Background(), policyRequest())
	if err != nil {
		t.Fatalf("ListSnapshotPolicies: %v", err)
	}
	var out struct {
		Unsupported bool   `json:"unsupported"`
		Warning     string `json:"warning"`
	}
	_ = json.Unmarshal(resp.Body, &out)
	if !out.Unsupported || out.Warning == "" {
		t.Errorf("payload = %s, want unsupported warning", resp.Body)
	}
}

func TestListSnapshotPoliciesUninitializedSM(t *testing.T) {
	fake := &fakePolicyAPI{
		snapshotResp: &domain.Response{Status: 404, Body: json.RawMessage(`{"error":{"type":"index_not_found_exception","reason":"no such index [.opendistro-sm-config]"}}`)},
	}
	svc := NewPoliciesService(&fakeClusterAPI{backendType: "opensearch"}, fake)
	resp, err := svc.ListSnapshotPolicies(context.Background(), policyRequest())
	if err != nil {
		t.Fatalf("ListSnapshotPolicies: %v", err)
	}
	if resp.Status != 200 {
		t.Fatalf("status = %d, want 200", resp.Status)
	}
	var out []domain.SnapshotPolicySummary
	if err := json.Unmarshal(resp.Body, &out); err != nil {
		t.Fatalf("unmarshal: %v (body=%s)", err, resp.Body)
	}
	if len(out) != 0 {
		t.Errorf("policies = %+v, want empty list", out)
	}
}
