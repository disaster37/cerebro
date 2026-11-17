package elastic

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"cerebro/internal/domain"
)

func newTestTarget(t *testing.T, url string, auth *domain.ESAuth, headers []domain.KeyValue, hostType string) domain.ElasticServer {
	t.Helper()
	return domain.ElasticServer{
		Host: domain.Host{
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
	target := newTestTarget(t, srv.URL, &domain.ESAuth{Username: "u", Password: "p"}, nil, "")
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
	target := newTestTarget(t, srv.URL, nil, []domain.KeyValue{
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

func newTLSTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	srv := httptest.NewUnstartedServer(handler)
	srv.Config.ErrorLog = log.New(io.Discard, "", 0)
	srv.StartTLS()
	return srv
}

func TestTargetTransportSelfSignedTLS(t *testing.T) {
	srv := newTLSTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tt := newTargetTransport(&http.Client{Timeout: 5 * time.Second}, false)
	target := newTestTarget(t, srv.URL, nil, nil, "")

	do := func(trust bool) (*http.Response, error) {
		target.TrustCertificate = trust
		req, _ := http.NewRequestWithContext(withTarget(context.Background(), target), http.MethodGet, "/", nil)
		return tt.RoundTrip(req)
	}

	if _, err := do(false); err == nil {
		t.Fatal("secure request to self-signed server succeeded, want TLS verification error")
	}

	resp, err := do(true)
	if err != nil {
		t.Fatalf("insecure request: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	if _, err := do(false); err == nil {
		t.Fatal("secure request reused insecure keep-alive connection, want TLS verification error")
	}
}

func TestTargetTransportHTTPWithTrustCertificateNoOp(t *testing.T) {
	cases := []struct {
		name string
		gzip bool
	}{
		{"gzip_true", true},
		{"gzip_false", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotAuth, gotHeader, gotAE string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				gotHeader = r.Header.Get("x-proxy-user")
				gotAE = r.Header.Get("Accept-Encoding")
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			tt := newTargetTransport(&http.Client{Timeout: 5 * time.Second}, tc.gzip)
			target := newTestTarget(t, srv.URL, &domain.ESAuth{Username: "u", Password: "p"}, []domain.KeyValue{
				{Key: "x-proxy-user", Value: "alice"},
			}, "")
			target.TrustCertificate = true
			req, _ := http.NewRequestWithContext(withTarget(context.Background(), target), http.MethodGet, "/", nil)
			resp, err := tt.RoundTrip(req)
			if err != nil {
				t.Fatalf("RoundTrip: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("status = %d, want 200", resp.StatusCode)
			}
			if !strings.HasPrefix(gotAuth, "Basic ") {
				t.Errorf("Authorization = %q, want Basic prefix", gotAuth)
			}
			if gotHeader != "alice" {
				t.Errorf("x-proxy-user = %q, want alice", gotHeader)
			}
			if tc.gzip {
				if gotAE != "" && gotAE != "gzip" {
					t.Errorf("Accept-Encoding = %q, want empty or gzip", gotAE)
				}
			} else if gotAE != "identity" {
				t.Errorf("Accept-Encoding = %q, want identity", gotAE)
			}
		})
	}
}

func TestTargetTransportConcurrentMixedTLS(t *testing.T) {
	srv := newTLSTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tt := newTargetTransport(&http.Client{Timeout: 5 * time.Second}, false)
	const n = 50
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		trust := i%2 == 0
		wg.Add(1)
		go func(trust bool) {
			defer wg.Done()
			target := newTestTarget(t, srv.URL, nil, nil, "")
			target.TrustCertificate = trust
			req, err := http.NewRequestWithContext(withTarget(context.Background(), target), http.MethodGet, "/", nil)
			if err != nil {
				errs <- err
				return
			}
			resp, err := tt.RoundTrip(req)
			if trust {
				if err != nil {
					errs <- err
					return
				}
				_ = resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					errs <- fmt.Errorf("insecure status = %d, want 200", resp.StatusCode)
				}
				return
			}
			if err == nil {
				_ = resp.Body.Close()
				errs <- fmt.Errorf("secure request to self-signed server succeeded, want TLS verification error")
			}
		}(trust)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
