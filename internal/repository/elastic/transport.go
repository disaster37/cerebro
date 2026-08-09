package elastic

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

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

func newTargetTransport(client *http.Client, gzip bool) *targetTransport {
	return &targetTransport{
		http: client,
		gzip: gzip,
	}
}

type targetTransport struct {
	http *http.Client
	gzip bool
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

	return t.http.Do(req)
}
