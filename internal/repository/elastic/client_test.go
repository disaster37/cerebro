package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cerebro/internal/domain"
)

// recordedRequest captures the last inbound request to a fake cluster.
type recordedRequest struct {
	method      string
	path        string
	rawQuery    string
	contentType string
	body        string
}

// fakeOSHandler speaks OpenSearch-style GET / (no X-Elastic-Product header)
// and records every inbound request.
func fakeOSHandler(rec *recordedRequest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rec.method = r.Method
		rec.path = r.URL.EscapedPath()
		rec.rawQuery = r.URL.RawQuery
		rec.contentType = r.Header.Get("Content-Type")
		if r.Body != nil {
			b, _ := io.ReadAll(r.Body)
			rec.body = string(b)
		}
		if r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/missing") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"index not found"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if r.URL.Path == "/" {
			_, _ = w.Write([]byte(`{"name":"opensearch","version":{"distribution":"opensearch","number":"3.4.0"}}`))
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/_nodes/stats/resource_usage_stats" {
			_, _ = w.Write([]byte(`{
				"_nodes":{"total":1,"successful":1,"failed":0},
				"cluster_name":"docker-cluster",
				"nodes":{
					"l5QPP9r8TL-0_Rb0IjBv6g":{
						"timestamp":1795453806252,
						"name":"node",
						"roles":["cluster_manager","data","ingest"],
						"resource_usage_stats":{
							"l5QPP9r8TL-0_Rb0IjBv6g":{
								"timestamp":1795453805391,
								"cpu_utilization_percent":"0.2",
								"memory_utilization_percent":"44.2",
								"native_memory_utilization_percent":"1.0",
								"io_usage_stats":{"max_io_utilization_percent":"0.0"}
							}
						}
					}
				}
			}`))
			return
		}
		_, _ = w.Write([]byte(`{"acknowledged":true}`))
	}
}

// fakeESHandler speaks ES-9-style GET / and sets the X-Elastic-Product:
// Elasticsearch header (so the v9 product check passes).
func fakeESHandler(rec *recordedRequest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rec.method = r.Method
		rec.path = r.URL.EscapedPath()
		rec.rawQuery = r.URL.RawQuery
		rec.contentType = r.Header.Get("Content-Type")
		if r.Body != nil {
			b, _ := io.ReadAll(r.Body)
			rec.body = string(b)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.WriteHeader(http.StatusOK)
		if r.URL.Path == "/" {
			_, _ = w.Write([]byte(`{"name":"es","version":{"number":"9.4.3"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"acknowledged":true}`))
	}
}

// newFakeOS returns a plain-HTTP OpenSearch fake.
func newFakeOS(t *testing.T, rec *recordedRequest) *httptest.Server {
	t.Helper()
	return httptest.NewServer(fakeOSHandler(rec))
}

// newFakeES returns a plain-HTTP Elasticsearch fake.
func newFakeES(t *testing.T, rec *recordedRequest) *httptest.Server {
	t.Helper()
	return httptest.NewServer(fakeESHandler(rec))
}

// newFakeOSTLS returns a self-signed TLS OpenSearch fake.
func newFakeOSTLS(t *testing.T, rec *recordedRequest) *httptest.Server {
	t.Helper()
	return newTLSTestServer(t, fakeOSHandler(rec))
}

// newFakeESTLS returns a self-signed TLS Elasticsearch fake.
func newFakeESTLS(t *testing.T, rec *recordedRequest) *httptest.Server {
	t.Helper()
	return newTLSTestServer(t, fakeESHandler(rec))
}

func targetFor(srv *httptest.Server, hostType string) domain.ElasticServer {
	return domain.ElasticServer{Host: domain.Host{Name: srv.URL, Type: hostType}}
}

func TestMultiClientDetectsAndDispatchesOpenSearch(t *testing.T) {
	var rec recordedRequest
	srv := newFakeOS(t, &rec)
	defer srv.Close()

	m, err := NewMultiClient(false)
	if err != nil {
		t.Fatalf("NewMultiClient: %v", err)
	}
	target := targetFor(srv, "")
	ctx := context.Background()

	resp, err := m.Main(ctx, target)
	if err != nil {
		t.Fatalf("Main: %v", err)
	}
	if resp.Status != 200 {
		t.Errorf("status = %d, want 200", resp.Status)
	}
	if !strings.Contains(string(resp.Body), `"distribution":"opensearch"`) {
		t.Errorf("body = %s, want OS body", resp.Body)
	}
	// fakeOS should have recorded GET / (detection) + GET / (Main).
	if rec.method != http.MethodGet || rec.path != "/" {
		t.Errorf("last recorded = %s %s, want GET /", rec.method, rec.path)
	}

	bi, err := m.BackendInfo(ctx, target)
	if err != nil {
		t.Fatalf("BackendInfo: %v", err)
	}
	if bi.Type != string(FlavorOpenSearch) {
		t.Errorf("BackendInfo.Type = %q, want opensearch", bi.Type)
	}
	if bi.Version != "3.4.0" {
		t.Errorf("BackendInfo.Version = %q, want 3.4.0", bi.Version)
	}
}

func TestMultiClientDetectsAndDispatchesElasticsearch(t *testing.T) {
	var rec recordedRequest
	srv := newFakeES(t, &rec)
	defer srv.Close()

	m, err := NewMultiClient(false)
	if err != nil {
		t.Fatalf("NewMultiClient: %v", err)
	}
	target := targetFor(srv, "")
	ctx := context.Background()

	resp, err := m.Main(ctx, target)
	if err != nil {
		t.Fatalf("Main: %v", err)
	}
	if resp.Status != 200 {
		t.Errorf("status = %d, want 200", resp.Status)
	}
	if !strings.Contains(string(resp.Body), `"number":"9.4.3"`) {
		t.Errorf("body = %s, want ES body", resp.Body)
	}

	bi, err := m.BackendInfo(ctx, target)
	if err != nil {
		t.Fatalf("BackendInfo: %v", err)
	}
	if bi.Type != string(FlavorElasticsearch) {
		t.Errorf("BackendInfo.Type = %q, want elasticsearch", bi.Type)
	}
	if bi.Version != "9.4.3" {
		t.Errorf("BackendInfo.Version = %q, want 9.4.3", bi.Version)
	}
}

func TestESEngineRejectsOpenSearch(t *testing.T) {
	var rec recordedRequest
	srv := newFakeOS(t, &rec)
	defer srv.Close()

	c, err := NewClient(false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	target := targetFor(srv, "")
	_, err = c.Main(context.Background(), target)
	if err == nil {
		t.Fatal("expected product-check error, got nil")
	}
	if !strings.Contains(err.Error(), "the client noticed that the server is not Elasticsearch") {
		t.Errorf("err = %q, want product-check message", err.Error())
	}
}

func TestConfiguredTypeWins(t *testing.T) {
	var rec recordedRequest
	srv := newFakeES(t, &rec)
	defer srv.Close()

	m, err := NewMultiClient(false)
	if err != nil {
		t.Fatalf("NewMultiClient: %v", err)
	}
	// Configured type opensearch overrides the sniffed ES flavor.
	target := targetFor(srv, "opensearch")
	ctx := context.Background()

	resp, err := m.Main(ctx, target)
	if err != nil {
		t.Fatalf("Main: %v", err)
	}
	if resp.Status != 200 {
		t.Errorf("status = %d, want 200", resp.Status)
	}
	// Dispatch should have used the OS engine (resty), which still hits fakeES.
	if rec.method != http.MethodGet || rec.path != "/" {
		t.Errorf("last recorded = %s %s, want GET /", rec.method, rec.path)
	}

	bi, err := m.BackendInfo(ctx, target)
	if err != nil {
		t.Fatalf("BackendInfo: %v", err)
	}
	if bi.Type != string(FlavorOpenSearch) {
		t.Errorf("BackendInfo.Type = %q, want opensearch (configured wins)", bi.Type)
	}
	// Detected version is kept even when configured type wins.
	if bi.Version != "9.4.3" {
		t.Errorf("BackendInfo.Version = %q, want 9.4.3 (detected)", bi.Version)
	}
}

func TestDetectionFailureFallsBackToElasticsearch(t *testing.T) {
	m, err := NewMultiClient(false)
	if err != nil {
		t.Fatalf("NewMultiClient: %v", err)
	}
	// Closed port: detection fails, falls back to ES engine, which then fails
	// to connect (transport error surfaces to the caller).
	target := domain.ElasticServer{Host: domain.Host{Name: "http://127.0.0.1:1"}}
	ctx := context.Background()

	info, derr := m.flavor(ctx, target)
	if derr == nil {
		t.Fatal("expected detection error, got nil")
	}
	if info.Flavor != FlavorElasticsearch {
		t.Errorf("fallback flavor = %q, want elasticsearch", info.Flavor)
	}

	_, err = m.Main(ctx, target)
	if err == nil {
		t.Fatal("expected transport error from ES engine, got nil")
	}
}

func TestOSEngineRequestBuilding(t *testing.T) {
	var rec recordedRequest
	srv := newFakeOS(t, &rec)
	defer srv.Close()

	m, err := NewMultiClient(false)
	if err != nil {
		t.Fatalf("NewMultiClient: %v", err)
	}
	os := m.os
	target := targetFor(srv, "")
	ctx := context.Background()

	cases := []struct {
		name      string
		call      func() (*domain.Response, error)
		method    string
		path      string
		query     string
		ct        string
		bodyCheck string
	}{
		{"Main", func() (*domain.Response, error) { return os.Main(ctx, target) }, "GET", "/", "", "", ""},
		{"ClusterHealth", func() (*domain.Response, error) { return os.ClusterHealth(ctx, target) }, "GET", "/_cluster/health", "", "", ""},
		{"IndexStats", func() (*domain.Response, error) { return os.IndexStats("idx", ctx, target) }, "GET", "/idx/_stats", "", "", ""},
		{"GetIndexSettingsFlat", func() (*domain.Response, error) { return os.GetIndexSettingsFlat("idx", ctx, target) }, "GET", "/idx/_settings", "flat_settings=true", "", ""},
		{"GetShardStats", func() (*domain.Response, error) { return os.GetShardStats("idx", ctx, target) }, "GET", "/idx/_stats", "level=shards", "", ""},
		{"CreateIndex", func() (*domain.Response, error) {
			return os.CreateIndex("idx", json.RawMessage(`{"settings":{}}`), ctx, target)
		}, "PUT", "/idx", "", "", `{"settings":{}}`},
		{"DeleteIndex", func() (*domain.Response, error) { return os.DeleteIndex("idx", ctx, target) }, "DELETE", "/idx", "", "", ""},
		{"UpdateAliases", func() (*domain.Response, error) {
			return os.UpdateAliases(json.RawMessage(`[{"add":{}}]`), ctx, target)
		}, "POST", "/_aliases", "", "", `{"actions":[`},
		{"GetIndices", func() (*domain.Response, error) { return os.GetIndices(ctx, target) }, "GET", "/_cat/indices", "format=json", "", ""},
		{"AnalyzeTextByField", func() (*domain.Response, error) { return os.AnalyzeTextByField("idx", "field1", "hello", ctx, target) }, "POST", "/idx/_analyze", "", "", `"field":"field1"`},
		{"CreateSnapshot", func() (*domain.Response, error) {
			return os.CreateSnapshot("repo", "snap", true, false, "idx", ctx, target)
		}, "PUT", "/_snapshot/repo/snap", "", "", `"ignore_unavailable":true`},
		{"RestoreSnapshot", func() (*domain.Response, error) {
			return os.RestoreSnapshot("repo", "snap", domain.RestoreSnapshotOptions{Indices: "idx", RenamePattern: "p", RenameReplacement: "r", IgnoreUnavailable: true, IncludeAliases: true}, ctx, target)
		}, "POST", "/_snapshot/repo/snap/_restore", "", "", `"rename_pattern":"p"`},
		{"CatRequest", func() (*domain.Response, error) { return os.CatRequest("indices", ctx, target) }, "GET", "/_cat/indices", "format=json", "", ""},
		{"ExecuteRequest_json", func() (*domain.Response, error) {
			return os.ExecuteRequest("GET", "/_cluster/settings", json.RawMessage(`{}`), ctx, target)
		}, "GET", "/_cluster/settings", "", "application/json", `{}`},
		{"ExecuteRequest_ndjson", func() (*domain.Response, error) {
			return os.ExecuteRequest("POST", "/_bulk", json.RawMessage(`{"index":{}}`+"\n"+`{"a":1}`+"\n"), ctx, target)
		}, "POST", "/_bulk", "", "application/x-ndjson", `{"index":{}`},
		{"CreateIndex_path_escape", func() (*domain.Response, error) {
			return os.CreateIndex("my index", json.RawMessage(`{}`), ctx, target)
		}, "PUT", "/my%20index", "", "", `{}`},
		{"ExecuteRequest_leading_slash", func() (*domain.Response, error) { return os.ExecuteRequest("GET", "_cluster/health", nil, ctx, target) }, "GET", "/_cluster/health", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec = recordedRequest{}
			if _, err := tc.call(); err != nil {
				t.Fatalf("call: %v", err)
			}
			if rec.method != tc.method {
				t.Errorf("method = %q, want %q", rec.method, tc.method)
			}
			if rec.path != tc.path {
				t.Errorf("path = %q, want %q", rec.path, tc.path)
			}
			if tc.query != "" && rec.rawQuery != tc.query {
				t.Errorf("rawQuery = %q, want %q", rec.rawQuery, tc.query)
			}
			if tc.ct != "" && rec.contentType != tc.ct {
				t.Errorf("content-type = %q, want %q", rec.contentType, tc.ct)
			}
			if tc.bodyCheck != "" && !strings.Contains(rec.body, tc.bodyCheck) {
				t.Errorf("body = %q, want to contain %q", rec.body, tc.bodyCheck)
			}
		})
	}
}

// TestESEngineExecuteRequestNormalizesLeadingSlash pins that the ES engine
// accepts REST-console paths without a leading slash (parity with the OS
// engine): a bare path is not a valid HTTP request target and would produce
// a malformed request.
func TestESEngineExecuteRequestNormalizesLeadingSlash(t *testing.T) {
	var rec recordedRequest
	srv := newFakeES(t, &rec)
	defer srv.Close()

	c, err := NewClient(false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := c.ExecuteRequest("GET", "_cluster/health", nil, context.Background(), targetFor(srv, ""))
	if err != nil {
		t.Fatalf("ExecuteRequest: %v", err)
	}
	if resp.Status != 200 {
		t.Errorf("status = %d, want 200", resp.Status)
	}
	if rec.path != "/_cluster/health" {
		t.Errorf("path = %q, want /_cluster/health", rec.path)
	}
}

func TestOSEngineNon2xxPassthrough(t *testing.T) {
	var rec recordedRequest
	srv := newFakeOS(t, &rec)
	defer srv.Close()

	m, err := NewMultiClient(false)
	if err != nil {
		t.Fatalf("NewMultiClient: %v", err)
	}
	os := m.os
	target := targetFor(srv, "")

	resp, err := os.DeleteIndex("missing", context.Background(), target)
	if err != nil {
		t.Fatalf("DeleteIndex: %v", err)
	}
	if resp.Status != 404 {
		t.Errorf("status = %d, want 404", resp.Status)
	}
	want := `{"error":"index not found"}`
	if !bytes.Equal(resp.Body, []byte(want)) {
		t.Errorf("body = %s, want %s (exact passthrough)", resp.Body, want)
	}
}

func TestMultiClientSelfSignedTLSOpenSearch(t *testing.T) {
	var rec recordedRequest
	srv := newFakeOSTLS(t, &rec)
	defer srv.Close()

	m, err := NewMultiClient(false)
	if err != nil {
		t.Fatalf("NewMultiClient: %v", err)
	}
	ctx := context.Background()
	target := targetFor(srv, "")

	if _, err := m.ClusterHealth(ctx, target); err == nil {
		t.Fatal("ClusterHealth with TrustCertificate=false succeeded, want TLS verification error")
	}
	if _, ok := m.flavors.get(strings.TrimRight(srv.URL, "/")); ok {
		t.Error("flavor cached after failed detection, want retry on next request")
	}

	target.TrustCertificate = true
	resp, err := m.ClusterHealth(ctx, target)
	if err != nil {
		t.Fatalf("ClusterHealth with TrustCertificate=true: %v", err)
	}
	if resp.Status != 200 {
		t.Errorf("status = %d, want 200", resp.Status)
	}

	bi, err := m.BackendInfo(ctx, target)
	if err != nil {
		t.Fatalf("BackendInfo: %v", err)
	}
	if bi.Type != string(FlavorOpenSearch) {
		t.Errorf("BackendInfo.Type = %q, want opensearch", bi.Type)
	}
	if bi.Version != "3.4.0" {
		t.Errorf("BackendInfo.Version = %q, want 3.4.0", bi.Version)
	}

	target.TrustCertificate = false
	if _, err := m.ClusterHealth(ctx, target); err == nil {
		t.Fatal("ClusterHealth with TrustCertificate=false succeeded after flavor cached, want TLS verification error")
	}
}

func TestMultiClientSelfSignedTLSElasticsearch(t *testing.T) {
	var rec recordedRequest
	srv := newFakeESTLS(t, &rec)
	defer srv.Close()

	m, err := NewMultiClient(false)
	if err != nil {
		t.Fatalf("NewMultiClient: %v", err)
	}
	ctx := context.Background()
	target := targetFor(srv, "")

	if _, err := m.ClusterHealth(ctx, target); err == nil {
		t.Fatal("ClusterHealth with TrustCertificate=false succeeded, want TLS verification error")
	}

	target.TrustCertificate = true
	resp, err := m.ClusterHealth(ctx, target)
	if err != nil {
		t.Fatalf("ClusterHealth with TrustCertificate=true: %v", err)
	}
	if resp.Status != 200 {
		t.Errorf("status = %d, want 200", resp.Status)
	}

	bi, err := m.BackendInfo(ctx, target)
	if err != nil {
		t.Fatalf("BackendInfo: %v", err)
	}
	if bi.Type != string(FlavorElasticsearch) {
		t.Errorf("BackendInfo.Type = %q, want elasticsearch", bi.Type)
	}
	if bi.Version != "9.4.3" {
		t.Errorf("BackendInfo.Version = %q, want 9.4.3", bi.Version)
	}
}

func TestOSEngineClusterStateMetricsAndCatMasterFallback(t *testing.T) {
	var rec recordedRequest
	handler := func(w http.ResponseWriter, r *http.Request) {
		rec.method = r.Method
		rec.path = r.URL.EscapedPath()
		rec.rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/" {
			_, _ = w.Write([]byte(`{"version":{"distribution":"opensearch","number":"3.4.0"}}`))
			return
		}
		if strings.Contains(r.URL.Path, "cluster_manager_node") || r.URL.Path == "/_cat/cluster_manager" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"unsupported"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}
	srv := httptest.NewServer(http.HandlerFunc(handler))
	defer srv.Close()

	m, err := NewMultiClient(false)
	if err != nil {
		t.Fatalf("NewMultiClient: %v", err)
	}
	target := targetFor(srv, "")
	ctx := context.Background()

	if _, err := m.os.ClusterStateMetrics(ctx, target); err != nil {
		t.Fatalf("ClusterStateMetrics: %v", err)
	}
	if rec.path != "/_cluster/state/master_node,routing_table,blocks" {
		t.Errorf("fallback path = %q, want master_node metrics", rec.path)
	}

	if _, err := m.os.CatMaster(ctx, target); err != nil {
		t.Fatalf("CatMaster: %v", err)
	}
	if rec.path != "/_cat/master" {
		t.Errorf("fallback path = %q, want /_cat/master", rec.path)
	}
}

func TestOSEngineNewEndpointPaths(t *testing.T) {
	var rec recordedRequest
	srv := newFakeOS(t, &rec)
	defer srv.Close()

	m, err := NewMultiClient(false)
	if err != nil {
		t.Fatalf("NewMultiClient: %v", err)
	}
	os := m.os
	target := targetFor(srv, "")
	ctx := context.Background()

	cases := []struct {
		name   string
		call   func() (*domain.Response, error)
		method string
		path   string
		query  string
		body   string
	}{
		{"ClusterStateMetrics", func() (*domain.Response, error) { return os.ClusterStateMetrics(ctx, target) }, "GET", "/_cluster/state/cluster_manager_node,routing_table,blocks", "", ""},
		{"CatMaster", func() (*domain.Response, error) { return os.CatMaster(ctx, target) }, "GET", "/_cat/cluster_manager", "format=json", ""},
		{"GetComposableTemplates", func() (*domain.Response, error) { return os.GetComposableTemplates(ctx, target) }, "GET", "/_index_template", "", ""},
		{"CreateComposableTemplate", func() (*domain.Response, error) {
			return os.CreateComposableTemplate("t", json.RawMessage(`{"index_patterns":["*"]}`), ctx, target)
		}, "PUT", "/_index_template/t", "", `"index_patterns"`},
		{"DeleteComposableTemplate", func() (*domain.Response, error) { return os.DeleteComposableTemplate("t", ctx, target) }, "DELETE", "/_index_template/t", "", ""},
		{"VerifyRepository", func() (*domain.Response, error) { return os.VerifyRepository("repo", ctx, target) }, "POST", "/_snapshot/repo/_verify", "", ""},
		{"GetSnapshotStatus", func() (*domain.Response, error) { return os.GetSnapshotStatus("repo", "_all", true, ctx, target) }, "GET", "/_snapshot/repo/_all/_status", "ignore_unavailable=true", ""},
		{"GetIndexLifecyclePolicies", func() (*domain.Response, error) { return os.GetIndexLifecyclePolicies(ctx, target) }, "GET", "/_plugins/_ism/policies", "", ""},
		{"ExplainIndexLifecycle", func() (*domain.Response, error) { return os.ExplainIndexLifecycle("*", ctx, target) }, "GET", "/_plugins/_ism/explain/*", "", ""},
		{"RetryIndexLifecycle", func() (*domain.Response, error) { return os.RetryIndexLifecycle([]string{"idx"}, "hot", ctx, target) }, "POST", "/_plugins/_ism/retry/idx", "", `"state":"hot"`},
		{"GetIndexLifecycleStatus", func() (*domain.Response, error) { return os.GetIndexLifecycleStatus(ctx, target) }, "GET", "/_cluster/settings", "flat_settings=true&include_defaults=true", ""},
		{"SetIndexLifecycleRunning", func() (*domain.Response, error) { return os.SetIndexLifecycleRunning(false, ctx, target) }, "PUT", "/_cluster/settings", "", `"plugins.index_state_management.enabled":false`},
		{"GetSnapshotLifecyclePolicies", func() (*domain.Response, error) { return os.GetSnapshotLifecyclePolicies(ctx, target) }, "GET", "/_plugins/_sm/policies", "", ""},
		{"ExplainSnapshotLifecyclePolicies", func() (*domain.Response, error) { return os.ExplainSnapshotLifecyclePolicies(ctx, target) }, "GET", "/_plugins/_sm/policies/*/_explain", "", ""},
		{"SnapshotPolicyStart", func() (*domain.Response, error) { return os.ExecuteOrToggleSnapshotPolicy("p1", "start", ctx, target) }, "POST", "/_plugins/_sm/policies/p1/_start", "", ""},
		{"SnapshotPolicyStop", func() (*domain.Response, error) { return os.ExecuteOrToggleSnapshotPolicy("p1", "stop", ctx, target) }, "POST", "/_plugins/_sm/policies/p1/_stop", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec = recordedRequest{}
			if _, err := tc.call(); err != nil {
				t.Fatalf("call: %v", err)
			}
			if rec.method != tc.method || rec.path != tc.path {
				t.Errorf("request = %s %s, want %s %s", rec.method, rec.path, tc.method, tc.path)
			}
			if tc.query != "" && rec.rawQuery != tc.query {
				t.Errorf("query = %q, want %q", rec.rawQuery, tc.query)
			}
			if tc.body != "" && !strings.Contains(rec.body, tc.body) {
				t.Errorf("body = %q, want to contain %q", rec.body, tc.body)
			}
		})
	}
}

func TestESEngineClusterStateMetricsPath(t *testing.T) {
	var rec recordedRequest
	srv := newFakeES(t, &rec)
	defer srv.Close()

	c, err := NewClient(false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := c.ClusterStateMetrics(context.Background(), targetFor(srv, "")); err != nil {
		t.Fatalf("ClusterStateMetrics: %v", err)
	}
	if rec.path != "/_cluster/state/master_node,routing_table,blocks" {
		t.Errorf("path = %q, want master_node metrics", rec.path)
	}
}

func TestOSEngineBodylessRequestsHaveNoContentType(t *testing.T) {
	var rec recordedRequest
	srv := newFakeOS(t, &rec)
	defer srv.Close()

	m, err := NewMultiClient(false)
	if err != nil {
		t.Fatalf("NewMultiClient: %v", err)
	}
	os := m.os
	target := targetFor(srv, "")
	ctx := context.Background()

	cases := []struct {
		name   string
		call   func() (*domain.Response, error)
		method string
		path   string
	}{
		{"Flush", func() (*domain.Response, error) { return os.FlushIndex("idx", ctx, target) }, "POST", "/idx/_flush"},
		{"Refresh", func() (*domain.Response, error) { return os.RefreshIndex("idx", ctx, target) }, "POST", "/idx/_refresh"},
		{"Open", func() (*domain.Response, error) { return os.OpenIndex("idx", ctx, target) }, "POST", "/idx/_open"},
		{"Close", func() (*domain.Response, error) { return os.CloseIndex("idx", ctx, target) }, "POST", "/idx/_close"},
		{"ForceMerge", func() (*domain.Response, error) { return os.ForceMerge("idx", ctx, target) }, "POST", "/idx/_forcemerge"},
		{"ClearCache", func() (*domain.Response, error) { return os.ClearIndexCache("idx", ctx, target) }, "POST", "/idx/_cache/clear"},
		{"Delete", func() (*domain.Response, error) { return os.DeleteIndex("idx", ctx, target) }, "DELETE", "/idx"},
		{"GetDataStreams", func() (*domain.Response, error) { return os.GetDataStreams(ctx, target) }, "GET", "/_data_stream"},
		{"CreateDataStream", func() (*domain.Response, error) { return os.CreateDataStream("logs-app", ctx, target) }, "PUT", "/_data_stream/logs-app"},
		{"DeleteDataStream", func() (*domain.Response, error) { return os.DeleteDataStream("logs-app", ctx, target) }, "DELETE", "/_data_stream/logs-app"},
		{"RolloverDataStream", func() (*domain.Response, error) { return os.RolloverDataStream("logs-app", ctx, target) }, "POST", "/logs-app/_rollover"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec = recordedRequest{}
			if _, err := tc.call(); err != nil {
				t.Fatalf("call: %v", err)
			}
			if rec.method != tc.method || rec.path != tc.path {
				t.Errorf("request = %s %s, want %s %s", rec.method, rec.path, tc.method, tc.path)
			}
			if rec.contentType != "" {
				t.Errorf("content-type = %q, want empty for bodyless request", rec.contentType)
			}
		})
	}
}

func TestOSEngineCommaSeparatedIndexActions(t *testing.T) {
	var rec recordedRequest
	srv := newFakeOS(t, &rec)
	defer srv.Close()

	m, err := NewMultiClient(false)
	if err != nil {
		t.Fatalf("NewMultiClient: %v", err)
	}
	os := m.os
	target := targetFor(srv, "")
	ctx := context.Background()

	cases := []struct {
		name   string
		call   func() (*domain.Response, error)
		method string
		path   string
	}{
		{"Flush", func() (*domain.Response, error) { return os.FlushIndex("idx1, idx2", ctx, target) }, "POST", "/idx1,idx2/_flush"},
		{"Refresh", func() (*domain.Response, error) { return os.RefreshIndex("idx1,idx2", ctx, target) }, "POST", "/idx1,idx2/_refresh"},
		{"Open", func() (*domain.Response, error) { return os.OpenIndex("idx1,idx2", ctx, target) }, "POST", "/idx1,idx2/_open"},
		{"Close", func() (*domain.Response, error) { return os.CloseIndex("idx1,idx2", ctx, target) }, "POST", "/idx1,idx2/_close"},
		{"ForceMerge", func() (*domain.Response, error) { return os.ForceMerge("idx1,idx2", ctx, target) }, "POST", "/idx1,idx2/_forcemerge"},
		{"ClearCache", func() (*domain.Response, error) { return os.ClearIndexCache("idx1,idx2", ctx, target) }, "POST", "/idx1,idx2/_cache/clear"},
		{"Delete", func() (*domain.Response, error) { return os.DeleteIndex("idx1,idx2", ctx, target) }, "DELETE", "/idx1,idx2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec = recordedRequest{}
			if _, err := tc.call(); err != nil {
				t.Fatalf("call: %v", err)
			}
			if rec.method != tc.method || rec.path != tc.path {
				t.Errorf("request = %s %s, want %s %s", rec.method, rec.path, tc.method, tc.path)
			}
		})
	}
}

func TestOSEngineAllocationExplain(t *testing.T) {
	var rec recordedRequest
	srv := newFakeOS(t, &rec)
	defer srv.Close()

	m, err := NewMultiClient(false)
	if err != nil {
		t.Fatalf("NewMultiClient: %v", err)
	}
	if _, err := m.os.AllocationExplain("idx", 0, true, context.Background(), targetFor(srv, "")); err != nil {
		t.Fatalf("AllocationExplain: %v", err)
	}
	if rec.method != "POST" || rec.path != "/_cluster/allocation/explain" {
		t.Errorf("request = %s %s", rec.method, rec.path)
	}
	if rec.contentType != "application/json" {
		t.Errorf("content-type = %q, want application/json", rec.contentType)
	}
	for _, want := range []string{`"index":"idx"`, `"shard":0`, `"primary":true`} {
		if !strings.Contains(rec.body, want) {
			t.Errorf("body = %q, want to contain %q", rec.body, want)
		}
	}
}

func TestESEngineNewEndpointPaths(t *testing.T) {
	var rec recordedRequest
	srv := newFakeES(t, &rec)
	defer srv.Close()

	c, err := NewClient(false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	target := targetFor(srv, "")
	ctx := context.Background()

	cases := []struct {
		name   string
		call   func() (*domain.Response, error)
		method string
		path   string
	}{
		{"FlushComma", func() (*domain.Response, error) { return c.FlushIndex("idx1,idx2", ctx, target) }, "POST", "/idx1,idx2/_flush"},
		{"DeleteComma", func() (*domain.Response, error) { return c.DeleteIndex("idx1,idx2", ctx, target) }, "DELETE", "/idx1,idx2"},
		{"GetDataStreams", func() (*domain.Response, error) { return c.GetDataStreams(ctx, target) }, "GET", "/_data_stream"},
		{"CreateDataStream", func() (*domain.Response, error) { return c.CreateDataStream("logs-app", ctx, target) }, "PUT", "/_data_stream/logs-app"},
		{"DeleteDataStream", func() (*domain.Response, error) { return c.DeleteDataStream("logs-app", ctx, target) }, "DELETE", "/_data_stream/logs-app"},
		{"RolloverDataStream", func() (*domain.Response, error) { return c.RolloverDataStream("logs-app", ctx, target) }, "POST", "/logs-app/_rollover"},
		{"AllocationExplain", func() (*domain.Response, error) { return c.AllocationExplain("idx", 0, false, ctx, target) }, "POST", "/_cluster/allocation/explain"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec = recordedRequest{}
			if _, err := tc.call(); err != nil {
				t.Fatalf("call: %v", err)
			}
			if rec.method != tc.method || rec.path != tc.path {
				t.Errorf("request = %s %s, want %s %s", rec.method, rec.path, tc.method, tc.path)
			}
		})
	}
}

type flatResourceUsageStats struct {
	Timestamp                      *int64   `json:"timestamp"`
	CPUUtilizationPercent          *float64 `json:"cpu_utilization_percent"`
	MemoryUtilizationPercent       *float64 `json:"memory_utilization_percent"`
	NativeMemoryUtilizationPercent *float64 `json:"native_memory_utilization_percent"`
	IOUsageStats                   *struct {
		MaxIOUtilizationPercent *float64 `json:"max_io_utilization_percent"`
	} `json:"io_usage_stats"`
}

type flatResourceUsageResponse struct {
	NodesStats struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Failed     int `json:"failed"`
	} `json:"_nodes"`
	ClusterName string `json:"cluster_name"`
	Nodes       map[string]struct {
		ResourceUsageStats flatResourceUsageStats `json:"resource_usage_stats"`
	} `json:"nodes"`
}

func decodeFlatResourceUsage(t *testing.T, body json.RawMessage) flatResourceUsageResponse {
	t.Helper()
	var out flatResourceUsageResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal normalized resource_usage_stats: %v\nbody: %s", err, body)
	}
	return out
}

func assertFloatValue(t *testing.T, label string, got *float64, want float64) {
	t.Helper()
	if got == nil {
		t.Errorf("%s = nil, want %v", label, want)
		return
	}
	if *got != want {
		t.Errorf("%s = %v, want %v", label, *got, want)
	}
}

func assertFloatNil(t *testing.T, label string, got *float64) {
	t.Helper()
	if got != nil {
		t.Errorf("%s = %v, want nil", label, *got)
	}
}

func floatPtr(v float64) *float64 { return &v }

func assertFloatPtrEqual(t *testing.T, label string, got, want *float64) {
	t.Helper()
	if want == nil {
		assertFloatNil(t, label, got)
		return
	}
	assertFloatValue(t, label, got, *want)
}

type wantCPUFields struct {
	source   string
	cpu      *float64
	resource *float64
}

func assertCPUFields(t *testing.T, label, source string, cpu, resource *float64, want wantCPUFields) {
	t.Helper()
	if source != want.source {
		t.Errorf("%s cpu_source = %q, want %q", label, source, want.source)
	}
	assertFloatPtrEqual(t, label+" cpu_percent", cpu, want.cpu)
	assertFloatPtrEqual(t, label+" resource_cpu_percent", resource, want.resource)
}

// fetchResourceUsage serves one canned GET /_nodes/stats/resource_usage_stats
// response so normalization edge cases can be exercised independently.
func fetchResourceUsage(t *testing.T, status int, body string) (*domain.Response, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/_nodes/stats/resource_usage_stats" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	m, err := NewMultiClient(false)
	if err != nil {
		t.Fatalf("NewMultiClient: %v", err)
	}
	return m.os.NodesResourceUsageStats(context.Background(), targetFor(srv, ""))
}

// resourceUsageDomainInputs builds minimal but complete nodes info/stats/state
// payloads whose os/process CPU sentinels are all unavailable, so the domain
// result reflects resource_usage_stats alone.
func resourceUsageDomainInputs(t *testing.T, ids ...string) (info, stats, state json.RawMessage) {
	t.Helper()
	if len(ids) == 0 {
		t.Fatal("resourceUsageDomainInputs requires at least one node ID")
	}
	infoNodes := make(map[string]any, len(ids))
	statsNodes := make(map[string]any, len(ids))
	for _, id := range ids {
		infoNodes[id] = map[string]any{
			"name":    id,
			"version": "3.10.0",
			"os":      map[string]any{"available_processors": 4},
			"jvm":     map[string]any{"version": "21"},
		}
		statsNodes[id] = map[string]any{
			"name":    id,
			"roles":   []string{"data"},
			"os":      map[string]any{"cpu": map[string]any{"percent": -1}},
			"process": map[string]any{"cpu": map[string]any{"percent": -1}},
			"jvm":     map[string]any{"mem": map[string]any{}},
			"fs":      map[string]any{"total": map[string]any{"total_in_bytes": 0}},
		}
	}
	mustMarshal := func(v any) json.RawMessage {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal domain input: %v", err)
		}
		return b
	}
	return mustMarshal(map[string]any{"nodes": infoNodes}),
		mustMarshal(map[string]any{"nodes": statsNodes}),
		mustMarshal(map[string]any{
			"cluster_manager_node": ids[0],
			"routing_table":        map[string]any{"indices": map[string]any{}},
			"blocks":               map[string]any{"indices": map[string]any{}},
		})
}

func nodesCPUByID(t *testing.T, resourceStats json.RawMessage, ids ...string) map[string]domain.NodePageEntry {
	t.Helper()
	info, stats, _ := resourceUsageDomainInputs(t, ids...)
	body, err := domain.NodesData(info, stats, json.RawMessage(`[]`), resourceStats)
	if err != nil {
		t.Fatalf("NodesData: %v", err)
	}
	var entries []domain.NodePageEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		t.Fatalf("unmarshal nodes body: %v", err)
	}
	out := make(map[string]domain.NodePageEntry, len(entries))
	for _, entry := range entries {
		out[entry.ID] = entry
	}
	return out
}

type overviewNodeCPU struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	CPUPercent         *float64 `json:"cpu_percent"`
	CPUSource          string   `json:"cpu_source"`
	ResourceCPUPercent *float64 `json:"resource_cpu_percent"`
}

func overviewCPUByID(t *testing.T, resourceStats json.RawMessage, ids ...string) map[string]overviewNodeCPU {
	t.Helper()
	info, stats, state := resourceUsageDomainInputs(t, ids...)
	body, err := domain.ClusterOverview(
		state, stats, json.RawMessage(`{}`), json.RawMessage(`{}`),
		json.RawMessage(`{}`), json.RawMessage(`{}`), info, nil, resourceStats,
	)
	if err != nil {
		t.Fatalf("ClusterOverview: %v", err)
	}
	var out struct {
		Nodes []overviewNodeCPU `json:"nodes"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal overview body: %v", err)
	}
	byID := make(map[string]overviewNodeCPU, len(out.Nodes))
	for _, node := range out.Nodes {
		byID[node.ID] = node
	}
	return byID
}

func TestOSEngineNodesResourceUsageStats(t *testing.T) {
	var rec recordedRequest
	srv := newFakeOS(t, &rec)
	defer srv.Close()

	m, err := NewMultiClient(false)
	if err != nil {
		t.Fatalf("NewMultiClient: %v", err)
	}
	resp, err := m.os.NodesResourceUsageStats(context.Background(), targetFor(srv, ""))
	if err != nil {
		t.Fatalf("NodesResourceUsageStats: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.Status)
	}
	if rec.method != http.MethodGet || rec.path != "/_nodes/stats/resource_usage_stats" {
		t.Errorf("request = %s %s, want GET /_nodes/stats/resource_usage_stats", rec.method, rec.path)
	}

	const nodeID = "l5QPP9r8TL-0_Rb0IjBv6g"
	flat := decodeFlatResourceUsage(t, resp.Body)
	node, ok := flat.Nodes[nodeID]
	if !ok {
		t.Fatalf("normalized nodes = %v, want %q", flat.Nodes, nodeID)
	}
	if flat.ClusterName != "docker-cluster" {
		t.Errorf("cluster_name = %q, want docker-cluster", flat.ClusterName)
	}
	if flat.NodesStats.Total != 1 || flat.NodesStats.Successful != 1 || flat.NodesStats.Failed != 0 {
		t.Errorf("_nodes = %+v, want total/successful/failed 1/1/0", flat.NodesStats)
	}
	usage := node.ResourceUsageStats
	if usage.Timestamp == nil || *usage.Timestamp != 1795453805391 {
		t.Errorf("timestamp = %v, want 1795453805391", usage.Timestamp)
	}
	assertFloatValue(t, "cpu_utilization_percent", usage.CPUUtilizationPercent, 0.2)
	assertFloatValue(t, "memory_utilization_percent", usage.MemoryUtilizationPercent, 44.2)
	assertFloatValue(t, "native_memory_utilization_percent", usage.NativeMemoryUtilizationPercent, 1.0)
	if usage.IOUsageStats == nil {
		t.Fatal("io_usage_stats = nil, want max_io_utilization_percent 0")
	}
	assertFloatValue(t, "max_io_utilization_percent", usage.IOUsageStats.MaxIOUtilizationPercent, 0)

	nodes := nodesCPUByID(t, resp.Body, nodeID)
	entry := nodes[nodeID]
	if entry.CPUSource != domain.CPUSourceResourceTracker {
		t.Errorf("nodes cpu_source = %q, want %q", entry.CPUSource, domain.CPUSourceResourceTracker)
	}
	assertFloatValue(t, "nodes cpu_percent", entry.CPUPercent, 0.2)
	assertFloatValue(t, "nodes resource_cpu_percent", entry.ResourceCPUPercent, 0.2)

	overview := overviewCPUByID(t, resp.Body, nodeID)[nodeID]
	if overview.CPUSource != domain.CPUSourceResourceTracker {
		t.Errorf("overview cpu_source = %q, want %q", overview.CPUSource, domain.CPUSourceResourceTracker)
	}
	assertFloatValue(t, "overview cpu_percent", overview.CPUPercent, 0.2)
	assertFloatValue(t, "overview resource_cpu_percent", overview.ResourceCPUPercent, 0.2)
}

func TestOSEngineNodesResourceUsageStatsNormalization(t *testing.T) {
	t.Run("multi-node mixed cluster and rounding", func(t *testing.T) {
		resp, err := fetchResourceUsage(t, http.StatusOK, `{
			"_nodes":{"total":6,"successful":6,"failed":0},
			"cluster_name":"docker-cluster",
			"nodes":{
				"n1":{"resource_usage_stats":{"n1":{"timestamp":111,"cpu_utilization_percent":"12.345","memory_utilization_percent":"60.5","native_memory_utilization_percent":"2.5","io_usage_stats":{"max_io_utilization_percent":"3.5"}}}},
				"n2":{"resource_usage_stats":{"n2":{"cpu_utilization_percent":"0.0"}}},
				"n3":{"resource_usage_stats":{"n3":{"cpu_utilization_percent":"-1","memory_utilization_percent":"51.5"}}},
				"n4":{"resource_usage_stats":{}},
				"n5":{},
				"n6":{"resource_usage_stats":null}
			}
		}`)
		if err != nil {
			t.Fatalf("NodesResourceUsageStats: %v", err)
		}

		flat := decodeFlatResourceUsage(t, resp.Body)
		if len(flat.Nodes) != 6 {
			t.Fatalf("nodes = %d, want 6", len(flat.Nodes))
		}
		n1 := flat.Nodes["n1"].ResourceUsageStats
		if n1.Timestamp == nil || *n1.Timestamp != 111 {
			t.Errorf("n1 timestamp = %v, want 111", n1.Timestamp)
		}
		assertFloatValue(t, "n1 cpu_utilization_percent", n1.CPUUtilizationPercent, 12.345)
		assertFloatValue(t, "n1 memory_utilization_percent", n1.MemoryUtilizationPercent, 60.5)
		assertFloatValue(t, "n1 native_memory_utilization_percent", n1.NativeMemoryUtilizationPercent, 2.5)
		if n1.IOUsageStats == nil {
			t.Fatal("n1 io_usage_stats = nil, want max_io_utilization_percent 3.5")
		}
		assertFloatValue(t, "n1 max_io_utilization_percent", n1.IOUsageStats.MaxIOUtilizationPercent, 3.5)
		assertFloatValue(t, "n2 cpu_utilization_percent", flat.Nodes["n2"].ResourceUsageStats.CPUUtilizationPercent, 0)
		assertFloatNil(t, "n3 cpu_utilization_percent", flat.Nodes["n3"].ResourceUsageStats.CPUUtilizationPercent)
		assertFloatValue(t, "n3 memory_utilization_percent", flat.Nodes["n3"].ResourceUsageStats.MemoryUtilizationPercent, 51.5)
		for _, id := range []string{"n4", "n5", "n6"} {
			assertFloatNil(t, id+" cpu_utilization_percent", flat.Nodes[id].ResourceUsageStats.CPUUtilizationPercent)
		}

		ids := []string{"n1", "n2", "n3", "n4", "n5", "n6"}
		want := map[string]wantCPUFields{
			"n1": {source: domain.CPUSourceResourceTracker, cpu: floatPtr(12.35), resource: floatPtr(12.35)},
			"n2": {source: domain.CPUSourceResourceTracker, cpu: floatPtr(0), resource: floatPtr(0)},
			"n3": {},
			"n4": {},
			"n5": {},
			"n6": {},
		}
		nodes := nodesCPUByID(t, resp.Body, ids...)
		overview := overviewCPUByID(t, resp.Body, ids...)
		for _, id := range ids {
			assertCPUFields(t, "nodes "+id, nodes[id].CPUSource, nodes[id].CPUPercent, nodes[id].ResourceCPUPercent, want[id])
			assertCPUFields(t, "overview "+id, overview[id].CPUSource, overview[id].CPUPercent, overview[id].ResourceCPUPercent, want[id])
		}
	})

	t.Run("node ID mismatch is never misattributed", func(t *testing.T) {
		resp, err := fetchResourceUsage(t, http.StatusOK, `{
			"nodes":{
				"n1":{"resource_usage_stats":{"n2":{"cpu_utilization_percent":"88.8"}}},
				"n2":{"resource_usage_stats":{"n1":{"cpu_utilization_percent":"88.8"},"n2":{"cpu_utilization_percent":"4.2"}}}
			}
		}`)
		if err != nil {
			t.Fatalf("NodesResourceUsageStats: %v", err)
		}
		flat := decodeFlatResourceUsage(t, resp.Body)
		assertFloatNil(t, "n1 cpu_utilization_percent", flat.Nodes["n1"].ResourceUsageStats.CPUUtilizationPercent)
		assertFloatValue(t, "n2 cpu_utilization_percent", flat.Nodes["n2"].ResourceUsageStats.CPUUtilizationPercent, 4.2)

		want := map[string]wantCPUFields{
			"n1": {},
			"n2": {source: domain.CPUSourceResourceTracker, cpu: floatPtr(4.2), resource: floatPtr(4.2)},
		}
		nodes := nodesCPUByID(t, resp.Body, "n1", "n2")
		overview := overviewCPUByID(t, resp.Body, "n1", "n2")
		for id, w := range want {
			assertCPUFields(t, "nodes "+id, nodes[id].CPUSource, nodes[id].CPUPercent, nodes[id].ResourceCPUPercent, w)
			assertCPUFields(t, "overview "+id, overview[id].CPUSource, overview[id].CPUPercent, overview[id].ResourceCPUPercent, w)
		}
	})

	t.Run("malformed non-finite and overflow values", func(t *testing.T) {
		resp, err := fetchResourceUsage(t, http.StatusOK, `{
			"nodes":{
				"empty":{"resource_usage_stats":{"empty":{"cpu_utilization_percent":""}}},
				"spaces":{"resource_usage_stats":{"spaces":{"cpu_utilization_percent":"   "}}},
				"text":{"resource_usage_stats":{"text":{"cpu_utilization_percent":"not-a-number"}}},
				"nan":{"resource_usage_stats":{"nan":{"cpu_utilization_percent":"NaN"}}},
				"inf":{"resource_usage_stats":{"inf":{"cpu_utilization_percent":"Inf"}}},
				"ninf":{"resource_usage_stats":{"ninf":{"cpu_utilization_percent":"-Inf"}}},
				"overflow":{"resource_usage_stats":{"overflow":{"cpu_utilization_percent":"1e500"}}},
				"rounding_overflow":{"resource_usage_stats":{"rounding_overflow":{"cpu_utilization_percent":"1e308"}}}
			}
		}`)
		if err != nil {
			t.Fatalf("NodesResourceUsageStats: %v", err)
		}
		flat := decodeFlatResourceUsage(t, resp.Body)
		ids := []string{"empty", "spaces", "text", "nan", "inf", "ninf", "overflow", "rounding_overflow"}
		if len(flat.Nodes) != len(ids) {
			t.Fatalf("nodes = %d, want %d", len(flat.Nodes), len(ids))
		}
		for _, id := range ids {
			assertFloatNil(t, id+" cpu_utilization_percent", flat.Nodes[id].ResourceUsageStats.CPUUtilizationPercent)
		}

		nodes := nodesCPUByID(t, resp.Body, ids...)
		overview := overviewCPUByID(t, resp.Body, ids...)
		for _, id := range ids {
			assertCPUFields(t, "nodes "+id, nodes[id].CPUSource, nodes[id].CPUPercent, nodes[id].ResourceCPUPercent, wantCPUFields{})
			assertCPUFields(t, "overview "+id, overview[id].CPUSource, overview[id].CPUPercent, overview[id].ResourceCPUPercent, wantCPUFields{})
		}
	})

	t.Run("finite burst above 100 percent", func(t *testing.T) {
		resp, err := fetchResourceUsage(t, http.StatusOK, `{
			"nodes":{"n1":{"resource_usage_stats":{"n1":{"cpu_utilization_percent":"125.678"}}}}
		}`)
		if err != nil {
			t.Fatalf("NodesResourceUsageStats: %v", err)
		}
		flat := decodeFlatResourceUsage(t, resp.Body)
		assertFloatValue(t, "n1 cpu_utilization_percent", flat.Nodes["n1"].ResourceUsageStats.CPUUtilizationPercent, 125.678)

		want := wantCPUFields{source: domain.CPUSourceResourceTracker, cpu: floatPtr(125.68), resource: floatPtr(125.68)}
		nodes := nodesCPUByID(t, resp.Body, "n1")
		overview := overviewCPUByID(t, resp.Body, "n1")
		assertCPUFields(t, "nodes n1", nodes["n1"].CPUSource, nodes["n1"].CPUPercent, nodes["n1"].ResourceCPUPercent, want)
		assertCPUFields(t, "overview n1", overview["n1"].CPUSource, overview["n1"].CPUPercent, overview["n1"].ResourceCPUPercent, want)
	})

	t.Run("non-2xx passthrough", func(t *testing.T) {
		cases := []struct {
			name   string
			status int
			body   string
		}{
			{"bad request", http.StatusBadRequest, `{"error":"unrecognized metric"}`},
			{"forbidden", http.StatusForbidden, `{"error":"forbidden"}`},
			{"server error", http.StatusInternalServerError, `{"error":"boom"}`},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				resp, err := fetchResourceUsage(t, tc.status, tc.body)
				if err != nil {
					t.Fatalf("NodesResourceUsageStats: %v", err)
				}
				if resp.Status != tc.status {
					t.Errorf("status = %d, want %d", resp.Status, tc.status)
				}
				if !bytes.Equal(resp.Body, []byte(tc.body)) {
					t.Errorf("body = %s, want exact passthrough %s", resp.Body, tc.body)
				}
			})
		}
	})

	t.Run("unexpected 2xx bodies fall back unchanged", func(t *testing.T) {
		cases := []struct {
			name     string
			body     string
			wantBody string
		}{
			{"missing nodes map", `{"acknowledged":true}`, `{"acknowledged":true}`},
			{"non-JSON body", `plain error`, `{"error":"plain error"}`},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				resp, err := fetchResourceUsage(t, http.StatusOK, tc.body)
				if err != nil {
					t.Fatalf("NodesResourceUsageStats: %v", err)
				}
				if resp.Status != http.StatusOK {
					t.Errorf("status = %d, want 200", resp.Status)
				}
				if !bytes.Equal(resp.Body, []byte(tc.wantBody)) {
					t.Errorf("body = %s, want unchanged %s", resp.Body, tc.wantBody)
				}
				nodes := nodesCPUByID(t, resp.Body, "n1")
				overview := overviewCPUByID(t, resp.Body, "n1")
				assertCPUFields(t, "nodes n1", nodes["n1"].CPUSource, nodes["n1"].CPUPercent, nodes["n1"].ResourceCPUPercent, wantCPUFields{})
				assertCPUFields(t, "overview n1", overview["n1"].CPUSource, overview["n1"].CPUPercent, overview["n1"].ResourceCPUPercent, wantCPUFields{})
			})
		}
	})
}

func TestESEngineNodesResourceUsageStatsSynthetic404WithoutIO(t *testing.T) {
	var rec recordedRequest
	srv := newFakeES(t, &rec)
	defer srv.Close()

	c, err := NewClient(false)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := c.NodesResourceUsageStats(context.Background(), targetFor(srv, ""))
	if err != nil {
		t.Fatalf("NodesResourceUsageStats: %v", err)
	}
	if resp.Status != http.StatusNotFound {
		t.Errorf("status = %d, want synthetic 404", resp.Status)
	}
	if rec.method != "" || rec.path != "" {
		t.Errorf("unexpected outbound request: %s %s", rec.method, rec.path)
	}
}
