package service

import (
	"context"
	"encoding/json"
	"testing"

	"cerebro/internal/domain"
)

type fakeSnapshotAPI struct {
	domain.SnapshotAPI
	verifyResp    *domain.Response
	verifyErr     error
	snapshotsResp *domain.Response
	statusResp    *domain.Response
	statusCalls   int
}

func (f *fakeSnapshotAPI) VerifyRepository(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return f.verifyResp, f.verifyErr
}

func (f *fakeSnapshotAPI) GetSnapshots(repository string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return f.snapshotsResp, nil
}

func (f *fakeSnapshotAPI) GetSnapshotStatus(repository, snapshot string, ignoreUnavailable bool, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	f.statusCalls++
	return f.statusResp, nil
}

func verifyRequest(t *testing.T, body string) domain.CerebroRequest {
	t.Helper()
	return domain.CerebroRequest{
		Target: domain.ElasticServer{Host: domain.Host{Name: "http://es:9200"}},
		Body:   json.RawMessage(body),
	}
}

func TestRepositoriesVerifyInlineSize(t *testing.T) {
	fake := &fakeSnapshotAPI{
		verifyResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"nodes":{"n1":{"name":"node-1"}}}`)},
		snapshotsResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"snapshots":[
			{"snapshot":"s1","state":"SUCCESS","stats":{"incremental":{"size_in_bytes":100},"total":{"size_in_bytes":300}}},
			{"snapshot":"s2","state":"SUCCESS","stats":{"incremental":{"size_in_bytes":50},"total":{"size_in_bytes":200}}}
		]}`)},
	}
	svc := NewRepositoriesService(fake)
	resp, err := svc.Verify(context.Background(), verifyRequest(t, `{"name":"repo"}`))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if resp.Status != 200 {
		t.Fatalf("status = %d, want 200", resp.Status)
	}
	var out domain.RepositoryVerifyStatus
	if err := json.Unmarshal(resp.Body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.Verified || out.Status != "CONNECTED" || out.NodeCount != 1 {
		t.Errorf("status = %+v", out)
	}
	if out.SnapshotCount != 2 || !out.SizeComputed || out.SizeInBytes == nil || *out.SizeInBytes != 150 {
		t.Errorf("size = %+v, want 150 computed", out)
	}
	if out.LogicalSizeInBytes == nil || *out.LogicalSizeInBytes != 300 {
		t.Errorf("logical size = %v, want 300", out.LogicalSizeInBytes)
	}
	if fake.statusCalls != 0 {
		t.Errorf("status calls = %d, want 0 (inline stats)", fake.statusCalls)
	}
}

func TestRepositoriesVerifyStatusFallback(t *testing.T) {
	fake := &fakeSnapshotAPI{
		verifyResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"nodes":{"n1":{"name":"node-1"}}}`)},
		snapshotsResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"snapshots":[
			{"snapshot":"s1","state":"SUCCESS"},
			{"snapshot":"s2","state":"PARTIAL"}
		]}`)},
		statusResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"snapshots":[
			{"snapshot":"s1","stats":{"incremental":{"size_in_bytes":100},"total":{"size_in_bytes":300}}},
			{"snapshot":"s2","stats":{"incremental":{"size_in_bytes":50},"total":{"size_in_bytes":200}}}
		]}`)},
	}
	svc := NewRepositoriesService(fake)
	resp, err := svc.Verify(context.Background(), verifyRequest(t, `{"name":"repo"}`))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	var out domain.RepositoryVerifyStatus
	_ = json.Unmarshal(resp.Body, &out)
	if fake.statusCalls != 1 {
		t.Errorf("status calls = %d, want 1", fake.statusCalls)
	}
	if out.SizeInBytes == nil || *out.SizeInBytes != 150 {
		t.Errorf("size = %v, want 150", out.SizeInBytes)
	}
	if out.FailedCount != 1 || out.Status != "PARTIAL_FAILURES" {
		t.Errorf("status = %+v, want PARTIAL_FAILURES with 1 failed", out)
	}
}

func TestRepositoriesVerifyErrorSurfacing(t *testing.T) {
	fake := &fakeSnapshotAPI{
		verifyResp: &domain.Response{Status: 500, Body: json.RawMessage(`{"error":{"type":"repository_exception","reason":"bucket not found"}}`)},
	}
	svc := NewRepositoriesService(fake)
	resp, err := svc.Verify(context.Background(), verifyRequest(t, `{"name":"repo"}`))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if resp.Status != 200 {
		t.Fatalf("status = %d, want 200 structured envelope", resp.Status)
	}
	var out domain.RepositoryVerifyStatus
	_ = json.Unmarshal(resp.Body, &out)
	if out.Verified || out.Status != "ERROR" || out.Error != "bucket not found" {
		t.Errorf("status = %+v, want ERROR with reason", out)
	}
}

func TestRepositoriesVerifySkipsLargeRepositories(t *testing.T) {
	snapshots := make([]map[string]any, 51)
	for i := range snapshots {
		snapshots[i] = map[string]any{"snapshot": "s", "state": "SUCCESS"}
	}
	body, _ := json.Marshal(map[string]any{"snapshots": snapshots})
	fake := &fakeSnapshotAPI{
		verifyResp:    &domain.Response{Status: 200, Body: json.RawMessage(`{"nodes":{}}`)},
		snapshotsResp: &domain.Response{Status: 200, Body: body},
	}
	svc := NewRepositoriesService(fake)
	resp, err := svc.Verify(context.Background(), verifyRequest(t, `{"name":"repo"}`))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	var out domain.RepositoryVerifyStatus
	_ = json.Unmarshal(resp.Body, &out)
	if out.SizeComputed || out.SizeSkippedReason == "" {
		t.Errorf("size = %+v, want skipped with reason", out)
	}
	if fake.statusCalls != 0 {
		t.Errorf("status calls = %d, want 0", fake.statusCalls)
	}
}
