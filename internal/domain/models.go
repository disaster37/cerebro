package domain

import (
	"encoding/json"
)

// ESAuth holds credentials for a target cluster.
type ESAuth struct {
	Username, Password string
}

// Host is a configured target cluster.
type Host struct {
	Name             string // target base URL (e.g. "http://es:9200")
	Type             string // configured flavor hint: "", "elasticsearch", "opensearch"
	Auth             *ESAuth
	HeadersWhitelist []string
}

// BackendInfo identifies the backend flavor/version of a target cluster for UI display.
type BackendInfo struct {
	Type    string `json:"backendType"`    // "elasticsearch" | "opensearch"
	Version string `json:"backendVersion"` // e.g. "9.4.3" / "3.4.0"; may be ""
}

// ElasticServer is a target cluster plus per-request connection options and
// forwarded (whitelisted) headers.
type ElasticServer struct {
	Host             Host
	Headers          []KeyValue
	TrustCertificate bool
}

// KeyValue is a generic header pair.
type KeyValue struct {
	Key, Value string
}

// User is the authenticated user (nil when auth disabled).
type User struct {
	Name string
}

// Response is the {status, body} envelope. HTTP status is always 200;
// the real ES status lives in Status. Body is the raw ES JSON (may be null).
type Response struct {
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

// NewResponse builds an envelope from an ES status + raw body.
func NewResponse(status int, body json.RawMessage) Response {
	return Response{Status: status, Body: body}
}

// NullResponse is the auth-redirect envelope: {"status":303,"body":null}.
func NullResponse() Response {
	return Response{Status: 303, Body: json.RawMessage("null")}
}

// ErrorResponse builds {"status":s,"body":{"error":msg}}.
func ErrorResponse(status int, msg string) Response {
	body, _ := json.Marshal(map[string]string{"error": msg})
	return Response{Status: status, Body: body}
}

// IsSuccess reports whether the ES/OS status carried by the envelope is 2xx.
func (r Response) IsSuccess() bool { return r.Status >= 200 && r.Status < 300 }

// RestoreSnapshotOptions is the full snapshot restore request contract shared
// by Elasticsearch 8/9 and OpenSearch 2/3.
type RestoreSnapshotOptions struct {
	Indices             string          `json:"indices"`
	RenamePattern       string          `json:"renamePattern"`
	RenameReplacement   string          `json:"renameReplacement"`
	IgnoreUnavailable   bool            `json:"ignoreUnavailable"`
	IncludeAliases      bool            `json:"includeAliases"`
	IncludeGlobalState  bool            `json:"includeGlobalState"`
	Partial             bool            `json:"partial"`
	IndexSettings       json.RawMessage `json:"indexSettings,omitempty"`
	IgnoreIndexSettings []string        `json:"ignoreIndexSettings,omitempty"`
}

// VerifiedRepoNode is one node that successfully verified a repository.
type VerifiedRepoNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// RepositoryVerifyStatus is the structured result of POST /repositories/verify.
type RepositoryVerifyStatus struct {
	Name               string             `json:"name"`
	Verified           bool               `json:"verified"`
	Status             string             `json:"status"` // "CONNECTED" | "IN_PROGRESS" | "PARTIAL_FAILURES" | "ERROR"
	NodeCount          int                `json:"node_count"`
	Nodes              []VerifiedRepoNode `json:"nodes"`
	SnapshotCount      int                `json:"snapshot_count"`
	ActiveCount        int                `json:"active_count"`
	FailedCount        int                `json:"failed_count"`
	SizeComputed       bool               `json:"size_computed"`
	SizeInBytes        *int64             `json:"size_in_bytes"`
	LogicalSizeInBytes *int64             `json:"logical_size_in_bytes"`
	SizeSkippedReason  string             `json:"size_skipped_reason,omitempty"`
	Error              string             `json:"error,omitempty"`
}
