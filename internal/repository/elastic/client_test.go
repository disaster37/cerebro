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
