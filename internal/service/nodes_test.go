package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"cerebro/internal/domain"
)

type fakeNodeAPI struct {
	domain.NodeAPI
	infoResp     *domain.Response
	statsResp    *domain.Response
	resourceResp *domain.Response
	resourceErr  error
	resourceCall int
}

func (f *fakeNodeAPI) Nodes(_ []string, _ context.Context, _ domain.ElasticServer) (*domain.Response, error) {
	return f.infoResp, nil
}

func (f *fakeNodeAPI) NodesStats(_ []string, _ context.Context, _ domain.ElasticServer) (*domain.Response, error) {
	return f.statsResp, nil
}

func (f *fakeNodeAPI) NodesResourceUsageStats(_ context.Context, _ domain.ElasticServer) (*domain.Response, error) {
	f.resourceCall++
	return f.resourceResp, f.resourceErr
}

type fakeCatAPI struct {
	domain.CatAPI
	masterResp *domain.Response
}

func (f *fakeCatAPI) CatMaster(_ context.Context, _ domain.ElasticServer) (*domain.Response, error) {
	return f.masterResp, nil
}

const nodesInfoJSON = `{"nodes":{"n1":{"name":"node-1","version":"3.4.0","jvm":{"version":"21"},"os":{"available_processors":4}}}}`
const nodesStatsJSON = `{"nodes":{"n1":{"name":"node-1","roles":["data"],"os":{"cpu":{"percent":-1}},"process":{"cpu":{"percent":-1}},"jvm":{"mem":{}},"fs":{"total":{"total_in_bytes":0}}}}}`

func newNodeService(resourceResp *domain.Response, resourceErr error) *NodesService {
	return NewNodesService(
		&fakeNodeAPI{
			infoResp:     &domain.Response{Status: http.StatusOK, Body: json.RawMessage(nodesInfoJSON)},
			statsResp:    &domain.Response{Status: http.StatusOK, Body: json.RawMessage(nodesStatsJSON)},
			resourceResp: resourceResp,
			resourceErr:  resourceErr,
		},
		&fakeCatAPI{masterResp: &domain.Response{Status: http.StatusOK, Body: json.RawMessage(`[{"id":"n1"}]`)}},
	)
}

func decodeNodes(t *testing.T, resp domain.Response) []domain.NodePageEntry {
	t.Helper()
	if resp.Status != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", resp.Status, resp.Body)
	}
	var entries []domain.NodePageEntry
	if err := json.Unmarshal(resp.Body, &entries); err != nil {
		t.Fatalf("unmarshal nodes: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("nodes = %d, want 1", len(entries))
	}
	return entries
}

func TestNodesServiceIndexUsesResourceUsageStats(t *testing.T) {
	svc := newNodeService(&domain.Response{
		Status: http.StatusOK,
		Body:   json.RawMessage(`{"nodes":{"n1":{"resource_usage_stats":{"cpu_utilization_percent":12.34}}}}`),
	}, nil)

	resp, err := svc.Index(context.Background(), domain.CerebroRequest{})
	if err != nil {
		t.Fatalf("Index: %v", err)
	}
	node := decodeNodes(t, resp)[0]
	if node.CPUPercent == nil || *node.CPUPercent != 12.34 {
		t.Errorf("cpu = %v, want 12.34", node.CPUPercent)
	}
	if node.CPUSource != domain.CPUSourceResourceTracker {
		t.Errorf("cpu_source = %q, want %q", node.CPUSource, domain.CPUSourceResourceTracker)
	}
}

func TestNodesServiceIndexFallsBackWhenResourceUsageStatsUnavailable(t *testing.T) {
	cases := []struct {
		name string
		resp *domain.Response
		err  error
	}{
		{"elasticsearch 404", &domain.Response{Status: http.StatusNotFound, Body: json.RawMessage(`{"error":"unsupported"}`)}, nil},
		{"openSearch 400", &domain.Response{Status: http.StatusBadRequest, Body: json.RawMessage(`{"error":"unrecognized metric"}`)}, nil},
		{"restricted 403", &domain.Response{Status: http.StatusForbidden, Body: json.RawMessage(`{"error":"forbidden"}`)}, nil},
		{"transport error", nil, errors.New("connection refused")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newNodeService(tc.resp, tc.err)
			resp, err := svc.Index(context.Background(), domain.CerebroRequest{})
			if err != nil {
				t.Fatalf("Index: %v", err)
			}
			node := decodeNodes(t, resp)[0]
			if node.CPUPercent != nil {
				t.Errorf("cpu = %v, want nil when all sources are -1", *node.CPUPercent)
			}
			if node.CPUSource != "" {
				t.Errorf("cpu_source = %q, want empty", node.CPUSource)
			}
			if node.ResourceCPUPercent != nil {
				t.Errorf("resource_cpu_percent = %v, want nil", node.ResourceCPUPercent)
			}
		})
	}
}
