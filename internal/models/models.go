package models

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

// ElasticServer is a target cluster plus forwarded (whitelisted) headers.
type ElasticServer struct {
	Host    Host
	Headers []KeyValue
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
