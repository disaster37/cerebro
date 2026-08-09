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

// newFakeOS returns an httptest server speaking OpenSearch-style GET / (no
// X-Elastic-Product header) and recording every inbound request.
func newFakeOS(t *testing.T, rec *recordedRequest) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
}

// newFakeES returns an httptest server speaking ES-9-style GET / and setting
// the X-Elastic-Product: Elasticsearch header (so the v9 product check passes).
func newFakeES(t *testing.T, rec *recordedRequest) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
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
			return os.RestoreSnapshot("repo", "snap", "p", "r", true, true, false, "idx", ctx, target)
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
