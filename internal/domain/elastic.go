package domain

import (
	"context"
	"encoding/json"
)

// ClusterAPI is cluster-level operations.
type ClusterAPI interface {
	Main(ctx context.Context, target ElasticServer) (*Response, error)
	ClusterState(ctx context.Context, target ElasticServer) (*Response, error)
	ClusterHealth(ctx context.Context, target ElasticServer) (*Response, error)
	ClusterSettings(ctx context.Context, target ElasticServer) (*Response, error)
	GetClusterSettingsFlat(ctx context.Context, target ElasticServer) (*Response, error)
	PutClusterSettings(body string, ctx context.Context, target ElasticServer) (*Response, error)
	SaveClusterSettings(settings json.RawMessage, ctx context.Context, target ElasticServer) (*Response, error)
	EnableShardAllocation(ctx context.Context, target ElasticServer) (*Response, error)
	DisableShardAllocation(kind string, ctx context.Context, target ElasticServer) (*Response, error)
	RelocateShard(shard int, index, from, to string, ctx context.Context, target ElasticServer) (*Response, error)
	GetClusterMapping(ctx context.Context, target ElasticServer) (*Response, error)
	BackendInfo(ctx context.Context, target ElasticServer) (BackendInfo, error)
}

// IndexAPI is index-level operations.
type IndexAPI interface {
	IndicesStats(ctx context.Context, target ElasticServer) (*Response, error)
	IndexStats(index string, ctx context.Context, target ElasticServer) (*Response, error)
	CloseIndex(index string, ctx context.Context, target ElasticServer) (*Response, error)
	OpenIndex(index string, ctx context.Context, target ElasticServer) (*Response, error)
	RefreshIndex(index string, ctx context.Context, target ElasticServer) (*Response, error)
	FlushIndex(index string, ctx context.Context, target ElasticServer) (*Response, error)
	ForceMerge(index string, ctx context.Context, target ElasticServer) (*Response, error)
	ClearIndexCache(index string, ctx context.Context, target ElasticServer) (*Response, error)
	DeleteIndex(index string, ctx context.Context, target ElasticServer) (*Response, error)
	CreateIndex(index string, metadata json.RawMessage, ctx context.Context, target ElasticServer) (*Response, error)
	GetIndices(ctx context.Context, target ElasticServer) (*Response, error)
	GetIndexSettings(index string, ctx context.Context, target ElasticServer) (*Response, error)
	GetIndexSettingsFlat(index string, ctx context.Context, target ElasticServer) (*Response, error)
	GetIndexMapping(index string, ctx context.Context, target ElasticServer) (*Response, error)
	GetIndexMetadata(index string, ctx context.Context, target ElasticServer) (*Response, error)
	GetShardStats(index string, ctx context.Context, target ElasticServer) (*Response, error)
	GetIndexRecovery(index string, ctx context.Context, target ElasticServer) (*Response, error)
	UpdateIndexSettings(index string, settings json.RawMessage, ctx context.Context, target ElasticServer) (*Response, error)
}

// AliasAPI is alias operations.
type AliasAPI interface {
	Aliases(ctx context.Context, target ElasticServer) (*Response, error)
	GetAliases(ctx context.Context, target ElasticServer) (*Response, error)
	UpdateAliases(changes json.RawMessage, ctx context.Context, target ElasticServer) (*Response, error)
}

// NodeAPI is node operations.
type NodeAPI interface {
	Nodes(flags []string, ctx context.Context, target ElasticServer) (*Response, error)
	NodesStats(stats []string, ctx context.Context, target ElasticServer) (*Response, error)
	NodeStats(node string, ctx context.Context, target ElasticServer) (*Response, error)
	GetNodes(ctx context.Context, target ElasticServer) (*Response, error)
}

// TemplateAPI is index-template operations.
type TemplateAPI interface {
	GetTemplates(ctx context.Context, target ElasticServer) (*Response, error)
	CreateTemplate(name string, template json.RawMessage, ctx context.Context, target ElasticServer) (*Response, error)
	DeleteTemplate(name string, ctx context.Context, target ElasticServer) (*Response, error)
}

// SnapshotAPI is snapshot/repository operations.
type SnapshotAPI interface {
	GetRepositories(ctx context.Context, target ElasticServer) (*Response, error)
	CreateRepository(name, repoType string, settings json.RawMessage, ctx context.Context, target ElasticServer) (*Response, error)
	DeleteRepository(name string, ctx context.Context, target ElasticServer) (*Response, error)
	GetSnapshots(repository string, ctx context.Context, target ElasticServer) (*Response, error)
	DeleteSnapshot(repository, snapshot string, ctx context.Context, target ElasticServer) (*Response, error)
	CreateSnapshot(repository, snapshot string, ignoreUnavailable, includeGlobalState bool, indices string, ctx context.Context, target ElasticServer) (*Response, error)
	RestoreSnapshot(repository, snapshot string, renamePattern, renameReplacement string, ignoreUnavailable, includeAliases, includeGlobalState bool, indices string, ctx context.Context, target ElasticServer) (*Response, error)
}

// AnalysisAPI is analyzer operations.
type AnalysisAPI interface {
	AnalyzeTextByField(index, field, text string, ctx context.Context, target ElasticServer) (*Response, error)
	AnalyzeTextByAnalyzer(index, analyzer, text string, ctx context.Context, target ElasticServer) (*Response, error)
}

// CatAPI is _cat operations.
type CatAPI interface {
	CatRequest(api string, ctx context.Context, target ElasticServer) (*Response, error)
	CatMaster(ctx context.Context, target ElasticServer) (*Response, error)
}

// RawAPI is the pass-through arbitrary request.
type RawAPI interface {
	ExecuteRequest(method, path string, data json.RawMessage, ctx context.Context, target ElasticServer) (*Response, error)
}

// ElasticClient is the full backend client capability, composed of the narrow
// capability interfaces above. Infrastructure (repository/elastic.MultiClient)
// implements it; services depend on the narrow interfaces they need.
type ElasticClient interface {
	ClusterAPI
	IndexAPI
	AliasAPI
	NodeAPI
	TemplateAPI
	SnapshotAPI
	AnalysisAPI
	CatAPI
	RawAPI
}
