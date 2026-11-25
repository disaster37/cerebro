package elastic

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"cerebro/internal/domain"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestFlavorCacheKeyIncludesConfiguredType(t *testing.T) {
	url := "http://es:9200"
	es := flavorCacheKey(domain.ElasticServer{Host: domain.Host{Name: url, Type: "elasticsearch"}})
	os := flavorCacheKey(domain.ElasticServer{Host: domain.Host{Name: url, Type: "opensearch"}})
	auto := flavorCacheKey(domain.ElasticServer{Host: domain.Host{Name: url}})
	if es == os {
		t.Fatalf("elasticsearch and opensearch cache keys collide: %q", es)
	}
	if es == auto || os == auto {
		t.Fatalf("configured-type keys collide with auto-detect key: es=%q os=%q auto=%q", es, os, auto)
	}
}

func TestFlavorCacheNotPoisonedByStaleRepopulation(t *testing.T) {
	// Reproduces the hot-reload stale-repopulation race: a request that resolved
	// the OLD snapshot (type=elasticsearch) repopulates the cache after
	// ClearFlavorCache; a subsequent NEW request (type=opensearch) must not be
	// served the stale elasticsearch flavor for the full 10-minute TTL.
	fake := roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"version":{"number":"9.4.3"}}`)),
		}, nil
	})

	m := &MultiClient{
		transport: newTargetTransport(&http.Client{Transport: fake}, false),
		flavors:   newFlavorCache(flavorTTL),
		log:       logrus.New(),
	}
	m.log.SetOutput(io.Discard)

	ctx := context.Background()
	oldTarget := domain.ElasticServer{Host: domain.Host{Name: "http://es:9200", Type: "elasticsearch"}}
	newTarget := domain.ElasticServer{Host: domain.Host{Name: "http://es:9200", Type: "opensearch"}}

	if info, err := m.flavor(ctx, oldTarget); err != nil || info.Flavor != FlavorElasticsearch {
		t.Fatalf("initial flavor = %+v, err = %v", info, err)
	}

	// Reload clears the cache; a stale in-flight request (still holding the old
	// type) then re-populates it.
	m.ClearFlavorCache()
	if _, err := m.flavor(ctx, oldTarget); err != nil {
		t.Fatalf("stale repopulate: %v", err)
	}

	if info, err := m.flavor(ctx, newTarget); err != nil || info.Flavor != FlavorOpenSearch {
		t.Fatalf("new target flavor = %+v, err = %v; stale cache poisoned hot reload", info, err)
	}
}

func TestParseFlavor(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		wantFlav  Flavor
		wantVer   string
		wantError bool
	}{
		{
			name:     "elasticsearch_9",
			body:     `{"name":"docker-cluster","cluster_name":"docker-cluster","version":{"number":"9.4.3","build_flavor":"default","type":"docker"},"tagline":"You Know, for Search"}`,
			wantFlav: FlavorElasticsearch,
			wantVer:  "9.4.3",
		},
		{
			name:     "opensearch_3",
			body:     `{"name":"opensearch","cluster_name":"docker-cluster","version":{"distribution":"opensearch","number":"3.4.0","build_flavor":"default"},"tagline":"You Know, for Search"}`,
			wantFlav: FlavorOpenSearch,
			wantVer:  "3.4.0",
		},
		{
			name:     "opensearch_case_insensitive",
			body:     `{"version":{"distribution":"OpenSearch","number":"3.4.0"}}`,
			wantFlav: FlavorOpenSearch,
			wantVer:  "3.4.0",
		},
		{
			name:      "malformed_json",
			body:      `not json`,
			wantError: true,
		},
		{
			name:      "missing_version",
			body:      `{"name":"x"}`,
			wantError: true,
		},
		{
			name:      "missing_number",
			body:      `{"version":{"distribution":"opensearch"}}`,
			wantError: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := parseFlavor([]byte(tc.body))
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error, got %+v", info)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseFlavor: %v", err)
			}
			if info.Flavor != tc.wantFlav {
				t.Errorf("flavor = %q, want %q", info.Flavor, tc.wantFlav)
			}
			if info.Version != tc.wantVer {
				t.Errorf("version = %q, want %q", info.Version, tc.wantVer)
			}
		})
	}
}

func TestNormalizeFlavor(t *testing.T) {
	cases := []struct {
		in   string
		want Flavor
	}{
		{"", ""},
		{"elasticsearch", FlavorElasticsearch},
		{"Elasticsearch", FlavorElasticsearch},
		{"opensearch", FlavorOpenSearch},
		{"OpenSearch", FlavorOpenSearch},
		{"mysql", ""},
	}
	for _, tc := range cases {
		if got := NormalizeFlavor(tc.in); got != tc.want {
			t.Errorf("NormalizeFlavor(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFlavorCacheSetGet(t *testing.T) {
	c := newFlavorCache(time.Minute)
	c.set("http://es:9200", FlavorInfo{Flavor: FlavorOpenSearch, Version: "3.4.0"})
	info, ok := c.get("http://es:9200")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if info.Flavor != FlavorOpenSearch || info.Version != "3.4.0" {
		t.Errorf("got %+v, want {opensearch, 3.4.0}", info)
	}
}

func TestFlavorCacheTTLExpiry(t *testing.T) {
	c := newFlavorCache(1 * time.Millisecond)
	c.set("http://es:9200", FlavorInfo{Flavor: FlavorElasticsearch, Version: "9.4.3"})
	time.Sleep(5 * time.Millisecond)
	if _, ok := c.get("http://es:9200"); ok {
		t.Fatal("expected cache miss after TTL expiry")
	}
}

func TestFlavorCacheConcurrent(t *testing.T) {
	c := newFlavorCache(time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c.set("k", FlavorInfo{Flavor: FlavorOpenSearch, Version: "3.4.0"})
			c.get("k")
		}(i)
	}
	wg.Wait()
}

func TestMultiClientClearFlavorCache(t *testing.T) {
	m, err := NewMultiClient(false)
	if err != nil {
		t.Fatalf("NewMultiClient: %v", err)
	}
	m.flavors.set("http://es:9200", FlavorInfo{Flavor: FlavorOpenSearch, Version: "3.4.0"})
	if _, ok := m.flavors.get("http://es:9200"); !ok {
		t.Fatal("expected cache hit before clear")
	}

	m.ClearFlavorCache()
	if _, ok := m.flavors.get("http://es:9200"); ok {
		t.Fatal("expected cache miss after ClearFlavorCache")
	}

	var nilClient *MultiClient
	nilClient.ClearFlavorCache()
}
