package service

import (
	"context"
	"encoding/json"
	"testing"

	"cerebro/internal/domain"
)

type fakeSnapshotAPI struct {
	domain.SnapshotAPI
	verifyResp      *domain.Response
	verifyErr       error
	snapshotsResp   *domain.Response
	statusResp      *domain.Response
	statusResponses map[string]*domain.Response
	statusCalls     int
	statusSnapshots []string
}

func (f *fakeSnapshotAPI) VerifyRepository(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return f.verifyResp, f.verifyErr
}

func (f *fakeSnapshotAPI) GetSnapshots(repository string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return f.snapshotsResp, nil
}

func (f *fakeSnapshotAPI) GetSnapshotStatus(repository, snapshot string, ignoreUnavailable bool, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	f.statusCalls++
	f.statusSnapshots = append(f.statusSnapshots, snapshot)
	if f.statusResponses != nil {
		return f.statusResponses[snapshot], nil
	}
	return f.statusResp, nil
}

func verifyRequest(t *testing.T, body string) domain.CerebroRequest {
	t.Helper()
	return domain.CerebroRequest{
		Target: domain.ElasticServer{Host: domain.Host{Name: "http://es:9200"}},
		Body:   json.RawMessage(body),
	}
}

func TestRepositoriesVerifyPrefersStatusEndpoint(t *testing.T) {
	fake := &fakeSnapshotAPI{
		verifyResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"nodes":{"n1":{"name":"node-1"}}}`)},
		snapshotsResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"snapshots":[
			{"snapshot":"s1","state":"SUCCESS","stats":{"incremental":{"size_in_bytes":100},"total":{"size_in_bytes":300}}},
			{"snapshot":"s2","state":"SUCCESS","stats":{"incremental":{"size_in_bytes":50},"total":{"size_in_bytes":200}}}
		]}`)},
		statusResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"snapshots":[
			{"snapshot":"s1","stats":{"incremental":{"size_in_bytes":100},"processed":{"size_in_bytes":100},"total":{"size_in_bytes":300}}},
			{"snapshot":"s2","stats":{"incremental":{"size_in_bytes":50},"processed":{"size_in_bytes":50},"total":{"size_in_bytes":200}}}
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
	if fake.statusCalls != 1 {
		t.Errorf("status calls = %d, want 1 (authoritative _status)", fake.statusCalls)
	}
}

func TestRepositoriesVerifyStatusFallbackToInline(t *testing.T) {
	fake := &fakeSnapshotAPI{
		verifyResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"nodes":{"n1":{"name":"node-1"}}}`)},
		snapshotsResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"snapshots":[
			{"snapshot":"s1","state":"SUCCESS","stats":{"incremental":{"size_in_bytes":100},"total":{"size_in_bytes":300}}},
			{"snapshot":"s2","state":"PARTIAL","stats":{"incremental":{"size_in_bytes":50},"total":{"size_in_bytes":200}}}
		]}`)},
		statusResp: &domain.Response{Status: 404, Body: json.RawMessage(`{"error":"not found"}`)},
	}
	svc := NewRepositoriesService(fake)
	resp, err := svc.Verify(context.Background(), verifyRequest(t, `{"name":"repo"}`))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	var out domain.RepositoryVerifyStatus
	_ = json.Unmarshal(resp.Body, &out)
	if fake.statusCalls != 2 {
		t.Errorf("status calls = %d, want 2 (_all then named)", fake.statusCalls)
	}
	if out.SizeInBytes == nil || *out.SizeInBytes != 150 {
		t.Errorf("size = %v, want 150", out.SizeInBytes)
	}
	if out.FailedCount != 1 || out.Status != "PARTIAL_FAILURES" {
		t.Errorf("status = %+v, want PARTIAL_FAILURES with 1 failed", out)
	}
}

func TestRepositoriesVerifyFlatSnapshotInfoStats(t *testing.T) {
	// Real ES SnapshotInfo uses flat processed_size_in_bytes/total_size_in_bytes
	// keys instead of the nested _status shape.
	fake := &fakeSnapshotAPI{
		verifyResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"nodes":{}}`)},
		snapshotsResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"snapshots":[
			{"snapshot":"s1","state":"SUCCESS","stats":{"processed_size_in_bytes":4096,"total_size_in_bytes":8192}}
		]}`)},
		statusResp: &domain.Response{Status: 404, Body: json.RawMessage(`{"error":"not found"}`)},
	}
	svc := NewRepositoriesService(fake)
	resp, err := svc.Verify(context.Background(), verifyRequest(t, `{"name":"repo"}`))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	var out domain.RepositoryVerifyStatus
	_ = json.Unmarshal(resp.Body, &out)
	if !out.SizeComputed || out.SizeInBytes == nil || *out.SizeInBytes != 4096 {
		t.Errorf("size = %+v, want 4096 from flat SnapshotInfo stats", out)
	}
	if out.LogicalSizeInBytes == nil || *out.LogicalSizeInBytes != 8192 {
		t.Errorf("logical size = %v, want 8192", out.LogicalSizeInBytes)
	}
}

func TestRepositoriesVerifyZeroIncrementalFallback(t *testing.T) {
	// A deleted base snapshot can leave only zero-incremental snapshots.
	fake := &fakeSnapshotAPI{
		verifyResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"nodes":{}}`)},
		snapshotsResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"snapshots":[
			{"snapshot":"s1","state":"SUCCESS","stats":{"incremental":{"size_in_bytes":0},"processed":{"size_in_bytes":2048},"total":{"size_in_bytes":4096}}}
		]}`)},
		statusResp: &domain.Response{Status: 404, Body: json.RawMessage(`{"error":"not found"}`)},
	}
	svc := NewRepositoriesService(fake)
	resp, err := svc.Verify(context.Background(), verifyRequest(t, `{"name":"repo"}`))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	var out domain.RepositoryVerifyStatus
	_ = json.Unmarshal(resp.Body, &out)
	if out.SizeInBytes == nil || *out.SizeInBytes != 2048 {
		t.Errorf("size = %v, want 2048 (processed fallback)", out.SizeInBytes)
	}
}

func TestRepositoriesVerifyEmptyRepository(t *testing.T) {
	fake := &fakeSnapshotAPI{
		verifyResp:    &domain.Response{Status: 200, Body: json.RawMessage(`{"nodes":{}}`)},
		snapshotsResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"snapshots":[]}`)},
	}
	svc := NewRepositoriesService(fake)
	resp, err := svc.Verify(context.Background(), verifyRequest(t, `{"name":"repo"}`))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	var out domain.RepositoryVerifyStatus
	_ = json.Unmarshal(resp.Body, &out)
	if !out.SizeComputed || out.SizeInBytes == nil || *out.SizeInBytes != 0 {
		t.Errorf("size = %+v, want computed 0 B", out)
	}
	if fake.statusCalls != 0 {
		t.Errorf("status calls = %d, want 0", fake.statusCalls)
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

func TestRepositoriesVerifyStatusAllWildcardEmptyFallsBackToNamed(t *testing.T) {
	// Elasticsearch 9 / OpenSearch 3 return {"snapshots":[]} for
	// GET /_snapshot/{repo}/_all/_status; the named form returns real stats.
	fake := &fakeSnapshotAPI{
		verifyResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"nodes":{}}`)},
		snapshotsResp: &domain.Response{Status: 200, Body: json.RawMessage(`{"snapshots":[
			{"snapshot":"s1","state":"SUCCESS"}
		]}`)},
		statusResponses: map[string]*domain.Response{
			"_all": {Status: 200, Body: json.RawMessage(`{"snapshots":[]}`)},
			"s1":   {Status: 200, Body: json.RawMessage(`{"snapshots":[{"snapshot":"s1","stats":{"incremental":{"size_in_bytes":4096},"total":{"size_in_bytes":8192}}}]}`)},
		},
	}
	svc := NewRepositoriesService(fake)
	resp, err := svc.Verify(context.Background(), verifyRequest(t, `{"name":"repo"}`))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	var out domain.RepositoryVerifyStatus
	_ = json.Unmarshal(resp.Body, &out)
	if out.SizeInBytes == nil || *out.SizeInBytes != 4096 {
		t.Errorf("size = %v, want 4096 from named _status", out.SizeInBytes)
	}
	if fake.statusCalls != 2 {
		t.Errorf("status calls = %d, want 2", fake.statusCalls)
	}
	if len(fake.statusSnapshots) != 2 || fake.statusSnapshots[0] != "_all" || fake.statusSnapshots[1] != "s1" {
		t.Errorf("status snapshots = %v, want [_all s1]", fake.statusSnapshots)
	}
}
