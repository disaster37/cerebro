package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/esapi"

	"cerebro/internal/domain"
)

type Client struct {
	es *elasticsearch.Client
}

var _ domain.ElasticClient = (*Client)(nil)

// NewClient builds a standalone esapi-backed engine.
//
// NOTE: currently only used by client_test.go; production builds the Client
// inside NewMultiClient.
func NewClient(gzip bool) (*Client, error) {
	transport := newTargetTransport(&http.Client{Timeout: 30 * time.Second}, gzip)
	es, err := elasticsearch.NewClient(elasticsearch.Config{ //nolint:staticcheck // SA1019: v9 deprecates Config/NewClient but keeps them fully functional; functional-options migration tracked separately
		Addresses:  []string{"http://placeholder.invalid:9200"},
		MaxRetries: 0,
		Transport:  transport,
	})
	if err != nil {
		return nil, err
	}
	return &Client{es: es}, nil
}

func (c *Client) Main(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.InfoRequest{}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

// BackendInfo reports the Elasticsearch flavor statically (the router only
// selects this engine for Elasticsearch targets).
func (c *Client) BackendInfo(ctx context.Context, target domain.ElasticServer) (domain.BackendInfo, error) {
	return domain.BackendInfo{Type: string(FlavorElasticsearch)}, nil
}

func (c *Client) ClusterState(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.ClusterStateRequest{}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) ClusterStateMetrics(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.ClusterStateRequest{Metric: []string{"master_node", "routing_table", "blocks"}}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) IndicesStats(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesStatsRequest{}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) NodesStats(stats []string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.NodesStatsRequest{Metric: stats}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

// NodesResourceUsageStats is unsupported on Elasticsearch. Return a synthetic
// 404 immediately (no network round-trip) so callers fall back to the MXBean
// os/process CPU sources without wasting a request or surfacing a 400.
func (c *Client) NodesResourceUsageStats(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return &domain.Response{
		Status: http.StatusNotFound,
		Body:   json.RawMessage(`{"error":"resource_usage_stats unsupported on elasticsearch"}`),
	}, nil
}

func (c *Client) NodeStats(node string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.NodesStatsRequest{NodeID: []string{node}}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) IndexStats(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesStatsRequest{Index: []string{index}}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) ClusterSettings(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.ClusterGetSettingsRequest{}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) Aliases(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesGetAliasRequest{}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) ClusterHealth(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.ClusterHealthRequest{}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) Nodes(flags []string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.NodesInfoRequest{Metric: flags}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) CloseIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesCloseRequest{Index: splitIndexList(index)}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) OpenIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesOpenRequest{Index: splitIndexList(index)}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) RefreshIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesRefreshRequest{Index: splitIndexList(index)}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) FlushIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesFlushRequest{Index: splitIndexList(index)}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) ForceMerge(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesForcemergeRequest{Index: splitIndexList(index)}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) ClearIndexCache(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesClearCacheRequest{Index: splitIndexList(index)}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) DeleteIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesDeleteRequest{Index: splitIndexList(index)}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) AllocationExplain(index string, shard int, primary bool, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	body := allocationExplainBody(index, shard, primary)
	return c.ExecuteRequest(http.MethodPost, "/_cluster/allocation/explain", json.RawMessage(body), ctx, target)
}

func (c *Client) GetDataStreams(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesGetDataStreamRequest{}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) CreateDataStream(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesCreateDataStreamRequest{Name: name}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) DeleteDataStream(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesDeleteDataStreamRequest{Name: []string{name}}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) RolloverDataStream(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesRolloverRequest{Alias: name}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) GetIndexSettings(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesGetSettingsRequest{Index: []string{index}}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) GetIndexSettingsFlat(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	t := true
	res, err := esapi.IndicesGetSettingsRequest{Index: []string{index}, FlatSettings: &t}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) GetIndexMapping(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesGetMappingRequest{Index: []string{index}}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) PutClusterSettings(body string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.ClusterPutSettingsRequest{Body: stringsReader(body)}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) EnableShardAllocation(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return c.PutClusterSettings(shardAllocationBody("all"), ctx, target)
}

func (c *Client) DisableShardAllocation(kind string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return c.PutClusterSettings(shardAllocationBody(kind), ctx, target)
}

func (c *Client) GetShardStats(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesStatsRequest{Index: []string{index}, Level: "shards"}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) RelocateShard(shard int, index, from, to string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	body := relocateShardBody(shard, index, from, to)
	res, err := esapi.ClusterRerouteRequest{Body: stringsReader(body)}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) GetIndexRecovery(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesRecoveryRequest{Index: []string{index}}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) GetClusterMapping(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesGetMappingRequest{}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) GetAliases(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return c.Aliases(ctx, target)
}

func (c *Client) UpdateAliases(changes json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	body := updateAliasesBody(changes)
	res, err := esapi.IndicesUpdateAliasesRequest{Body: stringsReader(body)}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) GetIndexMetadata(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.ClusterStateRequest{Metric: []string{"metadata/" + index}}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) CreateIndex(index string, metadata json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	var body io.Reader
	if len(metadata) > 0 {
		body = bytes.NewReader(metadata)
	}
	res, err := esapi.IndicesCreateRequest{Index: index, Body: body}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) GetIndices(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.CatIndicesRequest{Format: "json"}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) GetTemplates(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesGetTemplateRequest{}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) CreateTemplate(name string, template json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesPutTemplateRequest{Name: name, Body: bytes.NewReader(template)}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) DeleteTemplate(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesDeleteTemplateRequest{Name: name}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) GetComposableTemplates(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesGetIndexTemplateRequest{}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) CreateComposableTemplate(name string, template json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesPutIndexTemplateRequest{Name: name, Body: bytes.NewReader(template)}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) DeleteComposableTemplate(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesDeleteIndexTemplateRequest{Name: []string{name}}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) GetNodes(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.CatNodesRequest{Format: "json"}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) AnalyzeTextByField(index, field, text string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	body := analyzeByFieldBody(field, text)
	res, err := esapi.IndicesAnalyzeRequest{Index: index, Body: stringsReader(body)}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) AnalyzeTextByAnalyzer(index, analyzer, text string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	body := analyzeByAnalyzerBody(analyzer, text)
	res, err := esapi.IndicesAnalyzeRequest{Index: index, Body: stringsReader(body)}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) GetClusterSettingsFlat(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	t := true
	res, err := esapi.ClusterGetSettingsRequest{FlatSettings: &t, IncludeDefaults: &t}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) GetRepositories(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.SnapshotGetRepositoryRequest{}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) CreateRepository(name, repoType string, settings json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	body := createRepositoryBody(repoType, settings)
	res, err := esapi.SnapshotCreateRepositoryRequest{Repository: name, Body: stringsReader(body)}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) DeleteRepository(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.SnapshotDeleteRepositoryRequest{Repository: []string{name}}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) GetSnapshots(repository string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.SnapshotGetRequest{Repository: repository, Snapshot: []string{"_all"}}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) DeleteSnapshot(repository, snapshot string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.SnapshotDeleteRequest{Repository: repository, Snapshot: []string{snapshot}}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) CreateSnapshot(repository, snapshot string, ignoreUnavailable, includeGlobalState bool, indices string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	body := createSnapshotBody(ignoreUnavailable, includeGlobalState, indices)
	res, err := esapi.SnapshotCreateRequest{
		Repository: repository,
		Snapshot:   snapshot,
		Body:       stringsReader(body),
	}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) RestoreSnapshot(repository, snapshot string, opts domain.RestoreSnapshotOptions, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	body := restoreSnapshotBody(opts)
	res, err := esapi.SnapshotRestoreRequest{
		Repository: repository,
		Snapshot:   snapshot,
		Body:       stringsReader(body),
	}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) VerifyRepository(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.SnapshotVerifyRepositoryRequest{Repository: name}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) GetSnapshotStatus(repository, snapshot string, ignoreUnavailable bool, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	req := esapi.SnapshotStatusRequest{Repository: repository}
	if snapshot != "" {
		req.Snapshot = []string{snapshot}
	}
	if ignoreUnavailable {
		t := true
		req.IgnoreUnavailable = &t
	}
	res, err := req.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) GetIndexLifecyclePolicies(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.ILMGetLifecycleRequest{}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) ExplainIndexLifecycle(indexPattern string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	path := "/" + escapeIndexPattern(indexPattern) + "/_ilm/explain?only_managed=true&allow_no_indices=true&expand_wildcards=all"
	return c.ExecuteRequest(http.MethodGet, path, nil, ctx, target)
}

func (c *Client) RetryIndexLifecycle(indices []string, state string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	path := "/" + escapeIndexList(indices) + "/_ilm/retry"
	return c.ExecuteRequest(http.MethodPost, path, nil, ctx, target)
}

func (c *Client) GetIndexLifecycleStatus(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.ILMGetStatusRequest{}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) SetIndexLifecycleRunning(start bool, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	var res *esapi.Response
	var err error
	if start {
		res, err = esapi.ILMStartRequest{}.Do(withTarget(ctx, target), c.es)
	} else {
		res, err = esapi.ILMStopRequest{}.Do(withTarget(ctx, target), c.es)
	}
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) GetSnapshotLifecyclePolicies(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.SlmGetLifecycleRequest{}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) ExplainSnapshotLifecyclePolicies(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.SlmGetStatusRequest{}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) ExecuteOrToggleSnapshotPolicy(policyID, action string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	var res *esapi.Response
	var err error
	switch action {
	case "execute":
		res, err = esapi.SlmExecuteLifecycleRequest{PolicyID: policyID}.Do(withTarget(ctx, target), c.es)
	case "start":
		res, err = esapi.SlmStartRequest{}.Do(withTarget(ctx, target), c.es)
	case "stop":
		res, err = esapi.SlmStopRequest{}.Do(withTarget(ctx, target), c.es)
	default:
		resp := domain.ErrorResponse(400, "unsupported snapshot policy action: "+action)
		return &resp, nil
	}
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) SaveClusterSettings(settings json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.ClusterPutSettingsRequest{Body: bytes.NewReader(settings)}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) UpdateIndexSettings(index string, settings json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesPutSettingsRequest{Index: []string{index}, Body: bytes.NewReader(settings)}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) CatRequest(api string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	req, err := http.NewRequestWithContext(withTarget(ctx, target), http.MethodGet, "/_cat/"+api+"?format=json", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := c.es.Perform(req)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) CatMaster(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.CatMasterRequest{Format: "json"}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) ExecuteRequest(method, path string, data json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path // parity with the OS engine: a bare path is not a valid request target
	}
	var body io.Reader
	ct := "application/json"
	if len(data) > 0 {
		body = bytes.NewReader(data)
		if isNDJSON(data) {
			ct = "application/x-ndjson"
		}
	}
	req, err := http.NewRequestWithContext(withTarget(ctx, target), method, path, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", ct)
	}
	req.Header.Set("Accept", "application/json")
	res, err := c.es.Perform(req)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func stringsReader(s string) io.Reader {
	return bytes.NewReader([]byte(s))
}

// Shared body builders used by both the ES engine and the OS engine so that
// request bodies are byte-identical across flavors. All interpolated string
// values go through jsonStr so arbitrary user input always yields valid JSON
// (no JSON breakout/injection into the cluster request bodies).

// jsonStr marshals s as a JSON string literal (including the surrounding
// quotes), escaping quotes, backslashes and control characters.
func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func shardAllocationBody(enable string) string {
	return `{"transient":{"cluster":{"routing":{"allocation":{"enable":` + jsonStr(enable) + `}}}}}`
}

func relocateShardBody(shard int, index, from, to string) string {
	return `{"commands":[{"move":{"shard":` + strconv.Itoa(shard) +
		`,"index":` + jsonStr(index) +
		`,"from_node":` + jsonStr(from) +
		`,"to_node":` + jsonStr(to) + `}}]}`
}

func updateAliasesBody(changes json.RawMessage) string {
	return fmt.Sprintf(`{"actions":%s}`, changes)
}

func createRepositoryBody(repoType string, settings json.RawMessage) string {
	return `{"type":` + jsonStr(repoType) + `,"settings":` + string(settings) + `}`
}

func createSnapshotBody(ignoreUnavailable, includeGlobalState bool, indices string) string {
	body := fmt.Sprintf(`{"ignore_unavailable":%t,"include_global_state":%t`, ignoreUnavailable, includeGlobalState)
	if indices != "" {
		body += `,"indices":` + jsonStr(indices)
	}
	body += "}"
	return body
}

func restoreSnapshotBody(opts domain.RestoreSnapshotOptions) string {
	body := fmt.Sprintf(`{"ignore_unavailable":%t,"include_global_state":%t,"include_aliases":%t`, opts.IgnoreUnavailable, opts.IncludeGlobalState, opts.IncludeAliases)
	if opts.Indices != "" {
		body += `,"indices":` + jsonStr(opts.Indices)
	}
	if opts.RenamePattern != "" {
		body += `,"rename_pattern":` + jsonStr(opts.RenamePattern)
	}
	if opts.RenameReplacement != "" {
		body += `,"rename_replacement":` + jsonStr(opts.RenameReplacement)
	}
	if opts.Partial {
		body += `,"partial":true`
	}
	if len(opts.IndexSettings) > 0 && string(opts.IndexSettings) != "null" {
		body += `,"index_settings":` + string(opts.IndexSettings)
	}
	if len(opts.IgnoreIndexSettings) > 0 {
		settings, _ := json.Marshal(opts.IgnoreIndexSettings)
		body += `,"ignore_index_settings":` + string(settings)
	}
	body += "}"
	return body
}

// splitIndexList splits a comma-separated index list into individual names,
// trimming whitespace and dropping empty segments.
func splitIndexList(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func allocationExplainBody(index string, shard int, primary bool) string {
	return `{"index":` + jsonStr(index) +
		`,"shard":` + strconv.Itoa(shard) +
		`,"primary":` + strconv.FormatBool(primary) + `}`
}

// escapeIndexPattern escapes an index pattern for use as a single URL path
// segment while preserving the wildcard characters ES/OS accept in paths.
func escapeIndexPattern(pattern string) string {
	return strings.ReplaceAll(url.PathEscape(pattern), "%2A", "*")
}

// escapeIndexList escapes a comma-separated index list for a URL path.
func escapeIndexList(indices []string) string {
	escaped := make([]string, 0, len(indices))
	for _, idx := range indices {
		escaped = append(escaped, escapeIndexPattern(idx))
	}
	return strings.Join(escaped, ",")
}

func analyzeByFieldBody(field, text string) string {
	return `{"text":` + jsonStr(text) + `,"field":` + jsonStr(field) + `}`
}

func analyzeByAnalyzerBody(analyzer, text string) string {
	return `{"text":` + jsonStr(text) + `,"analyzer":` + jsonStr(analyzer) + `}`
}
