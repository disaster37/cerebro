package elastic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cerebro/internal/models"
)

func newTestTarget(t *testing.T, url string, auth *models.ESAuth, headers []models.KeyValue, hostType string) models.ElasticServer {
	t.Helper()
	return models.ElasticServer{
		Host: models.Host{
			Name: url,
			Type: hostType,
			Auth: auth,
		},
		Headers: headers,
	}
}

func TestTargetTransportRewritesHost(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tt := newTargetTransport(&http.Client{}, false)
	target := newTestTarget(t, srv.URL, nil, nil, "")
	req, _ := http.NewRequestWithContext(withTarget(context.Background(), target), http.MethodGet, "/_cluster/health?pretty=true", nil)
	resp, err := tt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if gotPath != "/_cluster/health" {
		t.Errorf("path = %q, want /_cluster/health", gotPath)
	}
	if gotQuery != "pretty=true" {
		t.Errorf("raw query = %q, want pretty=true", gotQuery)
	}
}

func TestTargetTransportBasicAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tt := newTargetTransport(&http.Client{}, false)
	target := newTestTarget(t, srv.URL, &models.ESAuth{Username: "u", Password: "p"}, nil, "")
	req, _ := http.NewRequestWithContext(withTarget(context.Background(), target), http.MethodGet, "/", nil)
	resp, err := tt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if !strings.HasPrefix(gotAuth, "Basic ") {
		t.Errorf("Authorization = %q, want Basic prefix", gotAuth)
	}
}

func TestTargetTransportForwardedHeaders(t *testing.T) {
	var xProxy, xExisting string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xProxy = r.Header.Get("x-proxy-user")
		xExisting = r.Header.Get("x-existing")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tt := newTargetTransport(&http.Client{}, false)
	target := newTestTarget(t, srv.URL, nil, []models.KeyValue{
		{Key: "x-proxy-user", Value: "alice"},
		{Key: "x-existing", Value: "should-not-overwrite"},
	}, "")
	req, _ := http.NewRequestWithContext(withTarget(context.Background(), target), http.MethodGet, "/", nil)
	req.Header.Set("x-existing", "original")
	resp, err := tt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if xProxy != "alice" {
		t.Errorf("x-proxy-user = %q, want alice", xProxy)
	}
	if xExisting != "original" {
		t.Errorf("x-existing = %q, want original (whitelisted header must not overwrite)", xExisting)
	}
}

func TestTargetTransportGzipRule(t *testing.T) {
	cases := []struct {
		name string
		gzip bool
		want string
	}{
		{"gzip_true_leaves_unset", true, ""},
		{"gzip_false_sets_identity", false, "identity"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotAE string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAE = r.Header.Get("Accept-Encoding")
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			tt := newTargetTransport(&http.Client{}, tc.gzip)
			target := newTestTarget(t, srv.URL, nil, nil, "")
			req, _ := http.NewRequestWithContext(withTarget(context.Background(), target), http.MethodGet, "/", nil)
			resp, err := tt.RoundTrip(req)
			if err != nil {
				t.Fatalf("RoundTrip: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if tc.want == "" {
				// gzip=true: header deleted; Go default transport may add "gzip" automatically.
				if gotAE != "" && gotAE != "gzip" {
					t.Errorf("Accept-Encoding = %q, want empty or gzip", gotAE)
				}
			} else {
				if gotAE != tc.want {
					t.Errorf("Accept-Encoding = %q, want %q", gotAE, tc.want)
				}
			}
		})
	}
}

func TestTargetTransportMissingTarget(t *testing.T) {
	tt := newTargetTransport(&http.Client{}, false)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	_, err := tt.RoundTrip(req)
	if err == nil {
		t.Fatal("expected error for missing target, got nil")
	}
}
