package elastic

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cerebro/internal/domain"
)

type targetKey struct{}

func withTarget(ctx context.Context, t domain.ElasticServer) context.Context {
	return context.WithValue(ctx, targetKey{}, t)
}

func targetFromContext(ctx context.Context) (domain.ElasticServer, bool) {
	t, ok := ctx.Value(targetKey{}).(domain.ElasticServer)
	return t, ok
}

func cloneDefaultTransport() *http.Transport {
	if dt, ok := http.DefaultTransport.(*http.Transport); ok {
		return dt.Clone()
	}
	return &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		ForceAttemptHTTP2: true,
	}
}

func newInsecureHTTPClient(timeout time.Duration) *http.Client {
	tr := cloneDefaultTransport()
	if tr.TLSClientConfig == nil {
		tr.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	} else {
		tr.TLSClientConfig = tr.TLSClientConfig.Clone()
	}
	// CWE-295: opt-in per-request TLS verification skip for self-signed cluster
	// certificates. Selected in RoundTrip strictly when target.TrustCertificate
	// is true; isolated in its own http.Transport so verified requests on t.http
	// never share pooled keep-alive TLS connections with unverified ones.
	tr.TLSClientConfig.InsecureSkipVerify = true //nolint:gosec // G402: explicit per-request user opt-in for self-signed cluster TLS certificates
	return &http.Client{
		Timeout:   timeout,
		Transport: tr,
	}
}

func newTargetTransport(client *http.Client, gzip bool) *targetTransport {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &targetTransport{
		http:         client,
		insecureHTTP: newInsecureHTTPClient(client.Timeout),
		gzip:         gzip,
	}
}

type targetTransport struct {
	http         *http.Client
	insecureHTTP *http.Client
	gzip         bool
}

func (t *targetTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	target, ok := targetFromContext(req.Context())
	if !ok {
		return nil, errors.New("missing ElasticServer target in request context")
	}

	base := strings.TrimRight(target.Host.Name, "/")
	u, err := url.Parse(base)
	if err != nil {
		u = &url.URL{Scheme: "http", Host: base}
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}
	req.URL.Scheme = u.Scheme
	req.URL.Host = u.Host
	req.Host = u.Host

	if target.Host.Auth != nil {
		req.SetBasicAuth(target.Host.Auth.Username, target.Host.Auth.Password)
	}

	for _, h := range target.Headers {
		if req.Header.Get(h.Key) == "" {
			req.Header.Set(h.Key, h.Value)
		}
	}

	if t.gzip {
		req.Header.Del("Accept-Encoding")
	} else {
		req.Header.Set("Accept-Encoding", "identity")
	}

	httpClient := t.http
	if target.TrustCertificate {
		httpClient = t.insecureHTTP
	}
	return httpClient.Do(req)
}
