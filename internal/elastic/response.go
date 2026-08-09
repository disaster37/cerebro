package elastic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"cerebro/internal/domain"
)

// isSuccess mirrors Scala ElasticResponse.isSuccess: 200..299.
func isSuccess(status int) bool {
	return status >= 200 && status < 300
}

// isNDJSON returns true when data is not a single valid JSON value. It mirrors
// the Scala executeRequest heuristic (a JsString body => ndjson/bulk/multi-
// search; any other JSON value => JSON): a bulk/multi-search body such as
// `{"index":{}}\n{"a":1}\n` is not a single JSON value, so it is classified as
// ndjson. Empty/whitespace-only data is treated as JSON (false).
func isNDJSON(data []byte) bool {
	if len(bytes.TrimSpace(data)) == 0 {
		return false
	}
	var v interface{}
	if json.Unmarshal(data, &v) == nil {
		return false // valid single JSON value -> JSON content type
	}
	return true // not a single JSON value -> ndjson (bulk / multi-search)
}

// toResponse converts an ES status + body reader into the envelope, parsing JSON
// body when valid and falling back to {"error": <body string>} when not.
// Always closes the body reader.
func toResponse(status int, body io.ReadCloser) (*domain.Response, error) {
	defer func() { _ = body.Close() }()
	raw, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var parsed json.RawMessage
	if err := json.Unmarshal(raw, &parsed); err != nil {
		// Not JSON — wrap the raw string in an error object (mirrors ES non-JSON
		// handling as {"error": <body string>}).
		msg, _ := json.Marshal(map[string]string{"error": string(raw)})
		return &domain.Response{Status: status, Body: msg}, nil
	}

	return &domain.Response{Status: status, Body: parsed}, nil
}
