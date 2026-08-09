package elastic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	opensearch "github.com/disaster37/opensearch/v4"
	"github.com/elastic/go-elasticsearch/v9"
	"github.com/sirupsen/logrus"

	"cerebro/internal/domain"
)

const placeholderURL = "http://placeholder.invalid:9200"

// MultiClient implements ElasticClient by routing each call to the ES engine
// (go-elasticsearch v9 / esapi) or the OS engine (disaster37/opensearch v4)
// based on the resolved flavor of the per-request target.
type MultiClient struct {
	es        *Client          // existing esapi-backed engine (unchanged internals)
	os        *osClient        // new resty-backed engine
	transport *targetTransport // shared by both engines and by detection
	flavors   *flavorCache
	log       *logrus.Logger
}

// NewMultiClient builds both engines over ONE shared targetTransport.
// gzip follows the es.gzip config (Accept-Encoding policy, base plan §3.4).
func NewMultiClient(gzip bool) (*MultiClient, error) {
	tt := newTargetTransport(&http.Client{Timeout: 30 * time.Second}, gzip)

	es, err := elasticsearch.NewClient(elasticsearch.Config{ //nolint:staticcheck // SA1019: deprecated in v9 but fully supported; functional-options migration is out of scope
		Addresses:  []string{placeholderURL},
		MaxRetries: 0,  // passthrough: never retry (preserve {status,body} of 429/5xx)
		Transport:  tt, // http.RoundTripper — unchanged in v9
	})
	if err != nil {
		return nil, err
	}

	// v4 client: placeholder base URL, NO creds (per-request auth lives in tt),
	// NO retries, default (Info-level) logger -> Debug hooks stay silent.
	osc, err := opensearch.New(&opensearch.Config{URL: placeholderURL}, logrus.NewEntry(logrus.StandardLogger()))
	if err != nil {
		return nil, err
	}
	// Config has no transport field: install the shared transport after New().
	// This replaces the cloned default transport built inside New() (its TLS /
	// idle-conn settings are discarded; tt owns all outbound policy).
	osc.RestyClient().SetTransport(tt)
	// resty drops request bodies on GET/HEAD by default; the ES engine sends
	// GET-with-body via http.NewRequestWithContext (e.g. GET _search with a
	// body), so enable GET payloads for parity.
	osc.RestyClient().SetAllowGetMethodPayload(true)

	return &MultiClient{
		es:        &Client{es: es},
		os:        &osClient{c: osc},
		transport: tt,
		flavors:   newFlavorCache(flavorTTL),
		log:       logrus.StandardLogger(),
	}, nil
}

// detect issues a raw GET / through the shared targetTransport directly — not
// through either client engine (avoids the go-elasticsearch product check on
// OS hosts and the v4 typed path).
func (m *MultiClient) detect(ctx context.Context, target domain.ElasticServer) (FlavorInfo, error) {
	req, err := http.NewRequestWithContext(withTarget(ctx, target), http.MethodGet, "/", nil)
	if err != nil {
		return FlavorInfo{}, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := m.transport.RoundTrip(req) // targetTransport: host rewrite, auth, headers, gzip
	if err != nil {
		return FlavorInfo{}, err
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return FlavorInfo{}, err
	}
	if !isSuccess(res.StatusCode) {
		return FlavorInfo{}, fmt.Errorf("flavor detection: unexpected status %d", res.StatusCode)
	}
	return parseFlavor(raw)
}

// flavor resolves the FlavorInfo for a target:
//  1. fresh cache hit -> return it
//  2. detect via GET /
//     - success: if target.Host.Type is configured, the CONFIGURED type wins
//     (operator intent) but the detected version is kept; cache and return
//     - failure: configured type -> return it (version "", NOT cached, so the
//     next request retries detection)
//     no configured type -> return FlavorElasticsearch fallback +
//     the detection error (caller logs a warning; NOT cached)
func (m *MultiClient) flavor(ctx context.Context, target domain.ElasticServer) (FlavorInfo, error) {
	key := strings.TrimRight(target.Host.Name, "/")
	if info, ok := m.flavors.get(key); ok {
		return info, nil
	}

	info, err := m.detect(ctx, target)
	if err == nil {
		// Configured type wins over sniffed distribution (operator override);
		// the detected version is still kept.
		if t := NormalizeFlavor(target.Host.Type); t != "" {
			info.Flavor = t
		}
		m.flavors.set(key, info)
		return info, nil
	}

	// Detection failure.
	if t := NormalizeFlavor(target.Host.Type); t != "" {
		// Configured type -> use it; NOT cached so the next request retries.
		return FlavorInfo{Flavor: t}, err
	}
	// No configured type -> ES fallback + the detection error (NOT cached).
	return FlavorInfo{Flavor: FlavorElasticsearch}, err
}

// isOpenSearch resolves flavor and returns true when the target is OpenSearch.
// Detection errors are logged as warnings (the caller proceeds with the
// resolved fallback flavor).
func (m *MultiClient) isOpenSearch(ctx context.Context, target domain.ElasticServer) bool {
	info, err := m.flavor(ctx, target)
	if err != nil {
		m.log.Warnf("flavor detection for %q failed: %v (using fallback %s)", target.Host.Name, err, info.Flavor)
	}
	return info.Flavor == FlavorOpenSearch
}

// BackendInfo implements the ElasticClient method used by /navbar (§6). It
// resolves the flavor (and version) of the target for UI display.
func (m *MultiClient) BackendInfo(ctx context.Context, target domain.ElasticServer) (domain.BackendInfo, error) {
	info, err := m.flavor(ctx, target)
	if err != nil {
		m.log.Warnf("BackendInfo detection for %q failed: %v (using fallback %s)", target.Host.Name, err, info.Flavor)
	}
	return domain.BackendInfo{Type: string(info.Flavor), Version: info.Version}, nil
}

// engine returns the appropriate ElasticClient for the target's resolved flavor.
// Both *Client and *osClient satisfy the ElasticClient interface.
func (m *MultiClient) engine(ctx context.Context, target domain.ElasticServer) ElasticClient {
	if m.isOpenSearch(ctx, target) {
		return m.os
	}
	return m.es
}

// Dispatch wrappers — one-liner per interface method. Each routes to the
// engine matching the resolved flavor of the per-request target.

func (m *MultiClient) Main(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).Main(ctx, target)
}

func (m *MultiClient) ClusterState(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).ClusterState(ctx, target)
}

func (m *MultiClient) IndicesStats(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).IndicesStats(ctx, target)
}

func (m *MultiClient) NodesStats(stats []string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).NodesStats(stats, ctx, target)
}

func (m *MultiClient) NodeStats(node string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).NodeStats(node, ctx, target)
}

func (m *MultiClient) IndexStats(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).IndexStats(index, ctx, target)
}

func (m *MultiClient) ClusterSettings(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).ClusterSettings(ctx, target)
}

func (m *MultiClient) Aliases(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).Aliases(ctx, target)
}

func (m *MultiClient) ClusterHealth(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).ClusterHealth(ctx, target)
}

func (m *MultiClient) Nodes(flags []string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).Nodes(flags, ctx, target)
}

func (m *MultiClient) CloseIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).CloseIndex(index, ctx, target)
}

func (m *MultiClient) OpenIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).OpenIndex(index, ctx, target)
}

func (m *MultiClient) RefreshIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).RefreshIndex(index, ctx, target)
}

func (m *MultiClient) FlushIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).FlushIndex(index, ctx, target)
}

func (m *MultiClient) ForceMerge(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).ForceMerge(index, ctx, target)
}

func (m *MultiClient) ClearIndexCache(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).ClearIndexCache(index, ctx, target)
}

func (m *MultiClient) DeleteIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).DeleteIndex(index, ctx, target)
}

func (m *MultiClient) GetIndexSettings(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).GetIndexSettings(index, ctx, target)
}

func (m *MultiClient) GetIndexSettingsFlat(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).GetIndexSettingsFlat(index, ctx, target)
}

func (m *MultiClient) GetIndexMapping(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).GetIndexMapping(index, ctx, target)
}

func (m *MultiClient) PutClusterSettings(body string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).PutClusterSettings(body, ctx, target)
}

func (m *MultiClient) EnableShardAllocation(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).EnableShardAllocation(ctx, target)
}

func (m *MultiClient) DisableShardAllocation(kind string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).DisableShardAllocation(kind, ctx, target)
}

func (m *MultiClient) GetShardStats(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).GetShardStats(index, ctx, target)
}

func (m *MultiClient) RelocateShard(shard int, index, from, to string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).RelocateShard(shard, index, from, to, ctx, target)
}

func (m *MultiClient) GetIndexRecovery(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).GetIndexRecovery(index, ctx, target)
}

func (m *MultiClient) GetClusterMapping(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).GetClusterMapping(ctx, target)
}

func (m *MultiClient) GetAliases(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).GetAliases(ctx, target)
}

func (m *MultiClient) UpdateAliases(changes json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).UpdateAliases(changes, ctx, target)
}

func (m *MultiClient) GetIndexMetadata(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).GetIndexMetadata(index, ctx, target)
}

func (m *MultiClient) CreateIndex(index string, metadata json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).CreateIndex(index, metadata, ctx, target)
}

func (m *MultiClient) GetIndices(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).GetIndices(ctx, target)
}

func (m *MultiClient) GetTemplates(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).GetTemplates(ctx, target)
}

func (m *MultiClient) CreateTemplate(name string, template json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).CreateTemplate(name, template, ctx, target)
}

func (m *MultiClient) DeleteTemplate(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).DeleteTemplate(name, ctx, target)
}

func (m *MultiClient) GetNodes(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).GetNodes(ctx, target)
}

func (m *MultiClient) AnalyzeTextByField(index, field, text string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).AnalyzeTextByField(index, field, text, ctx, target)
}

func (m *MultiClient) AnalyzeTextByAnalyzer(index, analyzer, text string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).AnalyzeTextByAnalyzer(index, analyzer, text, ctx, target)
}

func (m *MultiClient) GetClusterSettingsFlat(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).GetClusterSettingsFlat(ctx, target)
}

func (m *MultiClient) GetRepositories(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).GetRepositories(ctx, target)
}

func (m *MultiClient) CreateRepository(name, repoType string, settings json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).CreateRepository(name, repoType, settings, ctx, target)
}

func (m *MultiClient) DeleteRepository(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).DeleteRepository(name, ctx, target)
}

func (m *MultiClient) GetSnapshots(repository string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).GetSnapshots(repository, ctx, target)
}

func (m *MultiClient) DeleteSnapshot(repository, snapshot string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).DeleteSnapshot(repository, snapshot, ctx, target)
}

func (m *MultiClient) CreateSnapshot(repository, snapshot string, ignoreUnavailable, includeGlobalState bool, indices string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).CreateSnapshot(repository, snapshot, ignoreUnavailable, includeGlobalState, indices, ctx, target)
}

func (m *MultiClient) RestoreSnapshot(repository, snapshot string, renamePattern, renameReplacement string, ignoreUnavailable, includeAliases, includeGlobalState bool, indices string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).RestoreSnapshot(repository, snapshot, renamePattern, renameReplacement, ignoreUnavailable, includeAliases, includeGlobalState, indices, ctx, target)
}

func (m *MultiClient) SaveClusterSettings(settings json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).SaveClusterSettings(settings, ctx, target)
}

func (m *MultiClient) UpdateIndexSettings(index string, settings json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).UpdateIndexSettings(index, settings, ctx, target)
}

func (m *MultiClient) CatRequest(api string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).CatRequest(api, ctx, target)
}

func (m *MultiClient) CatMaster(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).CatMaster(ctx, target)
}

func (m *MultiClient) ExecuteRequest(method, path string, data json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return m.engine(ctx, target).ExecuteRequest(method, path, data, ctx, target)
}
