package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	opensearch "github.com/disaster37/opensearch/v4"

	"cerebro/internal/domain"
)

// osClient is the OpenSearch engine: every ElasticClient method is a RAW
// request through the v4 client's resty client whose method/path/params/body
// mirror what the esapi-backed Client sends (plan §5 table). No typed service
// calls are used: raw body + non-2xx passthrough would be lost, and every
// failure would be logged at Error level by the typed path (plan §2.6).
type osClient struct {
	c opensearch.Client // built via opensearch.New; transport swapped to targetTransport
}

var _ domain.ElasticClient = (*osClient)(nil)

// do executes one raw request against the per-request target and converts the
// result to the {status,body} envelope.
//
//   - resty returns err only for transport-level failures (no error hooks are
//     configured; non-2xx is NOT an error) -> err => handler maps to 502
//   - otherwise resp.StatusCode() + resp.Body() (raw bytes, may be empty)
//     become the envelope, including non-2xx (exact passthrough)
func (o *osClient) do(ctx context.Context, target domain.ElasticServer,
	method, path string, params url.Values, body, contentType string) (*domain.Response, error) {

	if !strings.HasPrefix(path, "/") {
		path = "/" + path // safe join with base URL
	}
	req := o.c.RestyClient().R().
		SetContext(withTarget(ctx, target)). // ctx value reaches targetTransport
		SetHeader("Accept", "application/json")
	if len(params) > 0 {
		req.SetQueryParamsFromValues(params)
	}
	if body != "" {
		req.SetBody(body) // string bodies sent verbatim (Content-Length set)
		if contentType == "" {
			contentType = "application/json"
		}
		req.SetHeader("Content-Type", contentType)
	}
	resp, err := req.Execute(method, path) // relative path joined to placeholderURL
	if err != nil {
		return nil, err
	}
	return toResponse(resp.StatusCode(), io.NopCloser(bytes.NewReader(resp.Body())))
}

// BackendInfo reports the OpenSearch flavor statically (the router only
// selects this engine for OpenSearch targets).
func (o *osClient) BackendInfo(ctx context.Context, target domain.ElasticServer) (domain.BackendInfo, error) {
	return domain.BackendInfo{Type: string(FlavorOpenSearch)}, nil
}

func (o *osClient) Main(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/", nil, "", "")
}

func (o *osClient) ClusterState(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/_cluster/state", nil, "", "")
}

func (o *osClient) ClusterStateMetrics(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	resp, err := o.do(ctx, target, http.MethodGet, "/_cluster/state/cluster_manager_node,routing_table,blocks", nil, "", "")
	if err == nil && resp.Status == http.StatusBadRequest {
		return o.do(ctx, target, http.MethodGet, "/_cluster/state/master_node,routing_table,blocks", nil, "", "")
	}
	return resp, err
}

func (o *osClient) IndicesStats(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/_stats", nil, "", "")
}

func (o *osClient) NodesStats(stats []string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/_nodes/stats/"+strings.Join(stats, ","), nil, "", "")
}

func (o *osClient) NodesResourceUsageStats(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/_nodes/stats/resource_usage_stats", nil, "", "")
}

func (o *osClient) NodeStats(node string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/_nodes/"+url.PathEscape(node)+"/stats", nil, "", "")
}

func (o *osClient) IndexStats(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/"+url.PathEscape(index)+"/_stats", nil, "", "")
}

func (o *osClient) ClusterSettings(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/_cluster/settings", nil, "", "")
}

func (o *osClient) Aliases(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/_alias", nil, "", "")
}

func (o *osClient) ClusterHealth(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/_cluster/health", nil, "", "")
}

func (o *osClient) Nodes(flags []string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/_nodes/"+strings.Join(flags, ","), nil, "", "")
}

func (o *osClient) CloseIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPost, "/"+escapeIndexList(splitIndexList(index))+"/_close", nil, "", "")
}

func (o *osClient) OpenIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPost, "/"+escapeIndexList(splitIndexList(index))+"/_open", nil, "", "")
}

func (o *osClient) RefreshIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPost, "/"+escapeIndexList(splitIndexList(index))+"/_refresh", nil, "", "")
}

func (o *osClient) FlushIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPost, "/"+escapeIndexList(splitIndexList(index))+"/_flush", nil, "", "")
}

func (o *osClient) ForceMerge(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPost, "/"+escapeIndexList(splitIndexList(index))+"/_forcemerge", nil, "", "")
}

func (o *osClient) ClearIndexCache(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPost, "/"+escapeIndexList(splitIndexList(index))+"/_cache/clear", nil, "", "")
}

func (o *osClient) DeleteIndex(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodDelete, "/"+escapeIndexList(splitIndexList(index)), nil, "", "")
}

func (o *osClient) AllocationExplain(index string, shard int, primary bool, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPost, "/_cluster/allocation/explain", nil, allocationExplainBody(index, shard, primary), "")
}

func (o *osClient) GetDataStreams(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/_data_stream", nil, "", "")
}

func (o *osClient) CreateDataStream(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPut, "/_data_stream/"+url.PathEscape(name), nil, "", "")
}

func (o *osClient) DeleteDataStream(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodDelete, "/_data_stream/"+url.PathEscape(name), nil, "", "")
}

func (o *osClient) RolloverDataStream(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPost, "/"+url.PathEscape(name)+"/_rollover", nil, "", "")
}

func (o *osClient) GetIndexSettings(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/"+url.PathEscape(index)+"/_settings", nil, "", "")
}

func (o *osClient) GetIndexSettingsFlat(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	params := url.Values{}
	params.Set("flat_settings", "true")
	return o.do(ctx, target, http.MethodGet, "/"+url.PathEscape(index)+"/_settings", params, "", "")
}

func (o *osClient) GetIndexMapping(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/"+url.PathEscape(index)+"/_mapping", nil, "", "")
}

func (o *osClient) PutClusterSettings(body string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPut, "/_cluster/settings", nil, body, "")
}

func (o *osClient) EnableShardAllocation(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.PutClusterSettings(shardAllocationBody("all"), ctx, target)
}

func (o *osClient) DisableShardAllocation(kind string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.PutClusterSettings(shardAllocationBody(kind), ctx, target)
}

func (o *osClient) GetShardStats(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	params := url.Values{}
	params.Set("level", "shards")
	return o.do(ctx, target, http.MethodGet, "/"+url.PathEscape(index)+"/_stats", params, "", "")
}

func (o *osClient) RelocateShard(shard int, index, from, to string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPost, "/_cluster/reroute", nil, relocateShardBody(shard, index, from, to), "")
}

func (o *osClient) GetIndexRecovery(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/"+url.PathEscape(index)+"/_recovery", nil, "", "")
}

func (o *osClient) GetClusterMapping(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/_mapping", nil, "", "")
}

func (o *osClient) GetAliases(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.Aliases(ctx, target)
}

func (o *osClient) UpdateAliases(changes json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPost, "/_aliases", nil, updateAliasesBody(changes), "")
}

func (o *osClient) GetIndexMetadata(index string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/_cluster/state/metadata/"+url.PathEscape(index), nil, "", "")
}

func (o *osClient) CreateIndex(index string, metadata json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPut, "/"+url.PathEscape(index), nil, string(metadata), "")
}

func (o *osClient) GetIndices(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	params := url.Values{}
	params.Set("format", "json")
	return o.do(ctx, target, http.MethodGet, "/_cat/indices", params, "", "")
}

func (o *osClient) GetTemplates(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/_template", nil, "", "")
}

func (o *osClient) CreateTemplate(name string, template json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPut, "/_template/"+url.PathEscape(name), nil, string(template), "")
}

func (o *osClient) DeleteTemplate(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodDelete, "/_template/"+url.PathEscape(name), nil, "", "")
}

func (o *osClient) GetComposableTemplates(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/_index_template", nil, "", "")
}

func (o *osClient) CreateComposableTemplate(name string, template json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPut, "/_index_template/"+url.PathEscape(name), nil, string(template), "")
}

func (o *osClient) DeleteComposableTemplate(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodDelete, "/_index_template/"+url.PathEscape(name), nil, "", "")
}

func (o *osClient) GetNodes(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	params := url.Values{}
	params.Set("format", "json")
	return o.do(ctx, target, http.MethodGet, "/_cat/nodes", params, "", "")
}

func (o *osClient) AnalyzeTextByField(index, field, text string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPost, "/"+url.PathEscape(index)+"/_analyze", nil, analyzeByFieldBody(field, text), "")
}

func (o *osClient) AnalyzeTextByAnalyzer(index, analyzer, text string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPost, "/"+url.PathEscape(index)+"/_analyze", nil, analyzeByAnalyzerBody(analyzer, text), "")
}

func (o *osClient) GetClusterSettingsFlat(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	params := url.Values{}
	params.Set("flat_settings", "true")
	params.Set("include_defaults", "true")
	return o.do(ctx, target, http.MethodGet, "/_cluster/settings", params, "", "")
}

func (o *osClient) GetRepositories(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/_snapshot", nil, "", "")
}

func (o *osClient) CreateRepository(name, repoType string, settings json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPut, "/_snapshot/"+url.PathEscape(name), nil, createRepositoryBody(repoType, settings), "")
}

func (o *osClient) DeleteRepository(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodDelete, "/_snapshot/"+url.PathEscape(name), nil, "", "")
}

func (o *osClient) GetSnapshots(repository string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/_snapshot/"+url.PathEscape(repository)+"/_all", nil, "", "")
}

func (o *osClient) DeleteSnapshot(repository, snapshot string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodDelete, "/_snapshot/"+url.PathEscape(repository)+"/"+url.PathEscape(snapshot), nil, "", "")
}

func (o *osClient) CreateSnapshot(repository, snapshot string, ignoreUnavailable, includeGlobalState bool, indices string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPut, "/_snapshot/"+url.PathEscape(repository)+"/"+url.PathEscape(snapshot), nil, createSnapshotBody(ignoreUnavailable, includeGlobalState, indices), "")
}

func (o *osClient) RestoreSnapshot(repository, snapshot string, opts domain.RestoreSnapshotOptions, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPost, "/_snapshot/"+url.PathEscape(repository)+"/"+url.PathEscape(snapshot)+"/_restore", nil, restoreSnapshotBody(opts), "")
}

func (o *osClient) VerifyRepository(name string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPost, "/_snapshot/"+url.PathEscape(name)+"/_verify", nil, "", "")
}

func (o *osClient) GetSnapshotStatus(repository, snapshot string, ignoreUnavailable bool, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	path := "/_snapshot/" + url.PathEscape(repository)
	if snapshot != "" {
		parts := splitIndexList(snapshot)
		escaped := make([]string, 0, len(parts))
		for _, part := range parts {
			escaped = append(escaped, url.PathEscape(part))
		}
		path += "/" + strings.Join(escaped, ",")
	}
	path += "/_status"
	params := url.Values{}
	if ignoreUnavailable {
		params.Set("ignore_unavailable", "true")
	}
	return o.do(ctx, target, http.MethodGet, path, params, "", "")
}

func (o *osClient) SaveClusterSettings(settings json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPut, "/_cluster/settings", nil, string(settings), "")
}

func (o *osClient) UpdateIndexSettings(index string, settings json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodPut, "/"+url.PathEscape(index)+"/_settings", nil, string(settings), "")
}

func (o *osClient) CatRequest(api string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	params := url.Values{}
	params.Set("format", "json")
	return o.do(ctx, target, http.MethodGet, "/_cat/"+api, params, "", "")
}

func (o *osClient) CatMaster(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	params := url.Values{}
	params.Set("format", "json")
	resp, err := o.do(ctx, target, http.MethodGet, "/_cat/cluster_manager", params, "", "")
	if err == nil && (resp.Status == http.StatusNotFound || resp.Status == http.StatusBadRequest) {
		return o.do(ctx, target, http.MethodGet, "/_cat/master", params, "", "")
	}
	return resp, err
}

func (o *osClient) GetIndexLifecyclePolicies(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/_plugins/_ism/policies", nil, "", "")
}

func (o *osClient) ExplainIndexLifecycle(indexPattern string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/_plugins/_ism/explain/"+escapeIndexPattern(indexPattern), nil, "", "")
}

func (o *osClient) RetryIndexLifecycle(indices []string, state string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	body := "{}"
	if state != "" {
		body = `{"state":` + jsonStr(state) + `}`
	}
	return o.do(ctx, target, http.MethodPost, "/_plugins/_ism/retry/"+escapeIndexList(indices), nil, body, "")
}

func (o *osClient) GetIndexLifecycleStatus(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	params := url.Values{}
	params.Set("include_defaults", "true")
	params.Set("flat_settings", "true")
	return o.do(ctx, target, http.MethodGet, "/_cluster/settings", params, "", "")
}

func (o *osClient) SetIndexLifecycleRunning(start bool, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	body := `{"persistent":{"plugins.index_state_management.enabled":` + strconv.FormatBool(start) + `}}`
	return o.do(ctx, target, http.MethodPut, "/_cluster/settings", nil, body, "")
}

func (o *osClient) GetSnapshotLifecyclePolicies(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/_plugins/_sm/policies", nil, "", "")
}

func (o *osClient) ExplainSnapshotLifecyclePolicies(ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	return o.do(ctx, target, http.MethodGet, "/_plugins/_sm/policies/*/_explain", nil, "", "")
}

func (o *osClient) ExecuteOrToggleSnapshotPolicy(policyID, action string, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	base := "/_plugins/_sm/policies/" + url.PathEscape(policyID)
	switch action {
	case "start":
		return o.do(ctx, target, http.MethodPost, base+"/_start", nil, "", "")
	case "stop":
		return o.do(ctx, target, http.MethodPost, base+"/_stop", nil, "", "")
	case "restart":
		if resp, err := o.do(ctx, target, http.MethodPost, base+"/_stop", nil, "", ""); err != nil || !resp.IsSuccess() {
			return resp, err
		}
		return o.do(ctx, target, http.MethodPost, base+"/_start", nil, "", "")
	default:
		resp := domain.ErrorResponse(400, "unsupported snapshot policy action: "+action)
		return &resp, nil
	}
}

func (o *osClient) ExecuteRequest(method, path string, data json.RawMessage, ctx context.Context, target domain.ElasticServer) (*domain.Response, error) {
	ct := "application/json"
	if len(data) > 0 && isNDJSON(data) {
		ct = "application/x-ndjson"
	}
	return o.do(ctx, target, method, path, nil, string(data), ct)
}
