package elastic

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestToResponsePassthrough(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		body     string
		wantStat int
		wantBody string
	}{
		{"200_json", 200, `{"cluster_name":"c","status":"green"}`, 200, `{"cluster_name":"c","status":"green"}`},
		{"404_json", 404, `{"error":"not found"}`, 404, `{"error":"not found"}`},
		{"non_json", 200, `plain text body`, 200, `{"error":"plain text body"}`},
		{"empty_body", 200, ``, 200, `{"error":""}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := io.NopCloser(strings.NewReader(tc.body))
			resp, err := toResponse(tc.status, r)
			if err != nil {
				t.Fatalf("toResponse: %v", err)
			}
			if resp.Status != tc.wantStat {
				t.Errorf("status = %d, want %d", resp.Status, tc.wantStat)
			}
			if !bytes.Equal(resp.Body, []byte(tc.wantBody)) {
				t.Errorf("body = %s, want %s", resp.Body, tc.wantBody)
			}
		})
	}
}

func TestIsSuccess(t *testing.T) {
	cases := []struct {
		status int
		want   bool
	}{
		{200, true}, {201, true}, {299, true}, {199, false}, {300, false}, {404, false}, {500, false},
	}
	for _, tc := range cases {
		if got := isSuccess(tc.status); got != tc.want {
			t.Errorf("isSuccess(%d) = %v, want %v", tc.status, got, tc.want)
		}
	}
}

func TestIsNDJSON(t *testing.T) {
	cases := []struct {
		name string
		data string
		want bool
	}{
		{"object", `{"a":1}`, false},
		{"array", `[1,2]`, false},
		{"string", `"s"`, false},
		{"number", `123`, false},
		{"true", `true`, false},
		{"null", `null`, false},
		{"leading_whitespace_object", `  {"a":1}`, false},
		{"bulk_ndjson", `{"index":{}}` + "\n" + `{"a":1}` + "\n", true},
		{"empty", ``, false},
		{"whitespace_only", `   `, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isNDJSON([]byte(tc.data)); got != tc.want {
				t.Errorf("isNDJSON(%q) = %v, want %v", tc.data, got, tc.want)
			}
		})
	}
}
