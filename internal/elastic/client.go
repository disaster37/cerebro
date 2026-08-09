package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/esapi"

	"cerebro/internal/domain"
)

type ElasticClient interface {
	Main(ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	ClusterState(ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	IndicesStats(ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	NodesStats(stats []string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	NodeStats(node string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	IndexStats(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	ClusterSettings(ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	Aliases(ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	ClusterHealth(ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	Nodes(flags []string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	CloseIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	OpenIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	RefreshIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	FlushIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	ForceMerge(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	ClearIndexCache(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	DeleteIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	GetIndexSettings(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	GetIndexSettingsFlat(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	GetIndexMapping(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	PutClusterSettings(body string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	EnableShardAllocation(ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	DisableShardAllocation(kind string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	GetShardStats(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	RelocateShard(shard int, index, from, to string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	GetIndexRecovery(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	GetClusterMapping(ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	GetAliases(ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	UpdateAliases(changes json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	GetIndexMetadata(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	CreateIndex(index string, metadata json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	GetIndices(ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	GetTemplates(ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	CreateTemplate(name string, template json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	DeleteTemplate(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	GetNodes(ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	AnalyzeTextByField(index, field, text string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	AnalyzeTextByAnalyzer(index, analyzer, text string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	GetClusterSettingsFlat(ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	GetRepositories(ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	CreateRepository(name, repoType string, settings json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	DeleteRepository(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	GetSnapshots(repository string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	DeleteSnapshot(repository, snapshot string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	CreateSnapshot(repository, snapshot string, ignoreUnavailable, includeGlobalState bool, indices string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	RestoreSnapshot(repository, snapshot string, renamePattern, renameReplacement string, ignoreUnavailable, includeAliases, includeGlobalState bool, indices string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	SaveClusterSettings(settings json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	UpdateIndexSettings(index string, settings json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	CatRequest(api string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	CatMaster(ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	ExecuteRequest(method, path string, data json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error)
	// BackendInfo returns the resolved backend flavor/version of the target.
	BackendInfo(ctx context.Context, target domain.ElasticServer) (domain.BackendInfo, error)
}

type Client struct {
	es *elasticsearch.Client
}

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
	res, err := esapi.IndicesCloseRequest{Index: []string{index}}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) OpenIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesOpenRequest{Index: []string{index}}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) RefreshIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesRefreshRequest{Index: []string{index}}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) FlushIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesFlushRequest{Index: []string{index}}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) ForceMerge(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesForcemergeRequest{Index: []string{index}}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) ClearIndexCache(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesClearCacheRequest{Index: []string{index}}.Do(withTarget(ctx, target), c.es)
	if err != nil {
		return nil, err
	}
	return toResponse(res.StatusCode, res.Body)
}

func (c *Client) DeleteIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	res, err := esapi.IndicesDeleteRequest{Index: []string{index}}.Do(withTarget(ctx, target), c.es)
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

func (c *Client) RestoreSnapshot(repository, snapshot string, renamePattern, renameReplacement string, ignoreUnavailable, includeAliases, includeGlobalState bool, indices string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	body := restoreSnapshotBody(renamePattern, renameReplacement, ignoreUnavailable, includeAliases, includeGlobalState, indices)
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

func restoreSnapshotBody(renamePattern, renameReplacement string, ignoreUnavailable, includeAliases, includeGlobalState bool, indices string) string {
	body := fmt.Sprintf(`{"ignore_unavailable":%t,"include_global_state":%t,"include_aliases":%t`, ignoreUnavailable, includeGlobalState, includeAliases)
	if indices != "" {
		body += `,"indices":` + jsonStr(indices)
	}
	if renamePattern != "" {
		body += `,"rename_pattern":` + jsonStr(renamePattern)
	}
	if renameReplacement != "" {
		body += `,"rename_replacement":` + jsonStr(renameReplacement)
	}
	body += "}"
	return body
}

func analyzeByFieldBody(field, text string) string {
	return `{"text":` + jsonStr(text) + `,"field":` + jsonStr(field) + `}`
}

func analyzeByAnalyzerBody(analyzer, text string) string {
	return `{"text":` + jsonStr(text) + `,"analyzer":` + jsonStr(analyzer) + `}`
}
