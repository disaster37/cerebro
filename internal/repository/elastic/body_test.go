package elastic

import (
	"encoding/json"
	"testing"
)

// TestBodyBuildersAlwaysProduceValidJSON pins the JSON-injection fix: no
// user-controlled value may break out of its JSON string and alter the
// request body structure sent to the cluster.
func TestBodyBuildersAlwaysProduceValidJSON(t *testing.T) {
	// Attempts to close the string/object and inject a new command.
	evil := `"}],"commands":[{"move":{"shard":0,"index":"pwned`

	bodies := map[string]string{
		"shardAllocation":   shardAllocationBody(evil),
		"relocateShard":     relocateShardBody(1, evil, evil, evil),
		"createSnapshot":    createSnapshotBody(true, false, evil),
		"restoreSnapshot":   restoreSnapshotBody(evil, evil, true, true, true, evil),
		"analyzeByField":    analyzeByFieldBody(evil, evil),
		"analyzeByAnalyzer": analyzeByAnalyzerBody(evil, evil),
		"createRepository":  createRepositoryBody(evil, json.RawMessage(`{}`)),
	}
	for name, body := range bodies {
		if !json.Valid([]byte(body)) {
			t.Errorf("%s: invalid JSON: %s", name, body)
		}
	}

	// The evil value must survive as DATA, not as structure.
	var m struct {
		Commands []struct {
			Move struct {
				Shard    int    `json:"shard"`
				Index    string `json:"index"`
				FromNode string `json:"from_node"`
				ToNode   string `json:"to_node"`
			} `json:"move"`
		} `json:"commands"`
	}
	body := relocateShardBody(7, evil, "nodeA", "nodeB")
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("relocateShardBody did not parse: %v\nbody: %s", err, body)
	}
	if len(m.Commands) != 1 {
		t.Fatalf("commands = %d, want exactly 1 (no injected command)", len(m.Commands))
	}
	if got := m.Commands[0].Move; got.Shard != 7 || got.Index != evil || got.FromNode != "nodeA" || got.ToNode != "nodeB" {
		t.Errorf("move = %+v, want shard 7 / evil index / nodeA / nodeB", got)
	}
}

// TestBodyBuildersStableForPlainValues pins that plain inputs produce the
// exact same bodies as before the escaping fix (behavior preservation).
func TestBodyBuildersStableForPlainValues(t *testing.T) {
	cases := map[string]string{
		shardAllocationBody("all"):                                         `{"transient":{"cluster":{"routing":{"allocation":{"enable":"all"}}}}}`,
		relocateShardBody(1, "i", "f", "t"):                                `{"commands":[{"move":{"shard":1,"index":"i","from_node":"f","to_node":"t"}}]}`,
		createRepositoryBody("fs", json.RawMessage(`{"location":"/tmp"}`)): `{"type":"fs","settings":{"location":"/tmp"}}`,
		createSnapshotBody(true, false, "idx"):                             `{"ignore_unavailable":true,"include_global_state":false,"indices":"idx"}`,
		analyzeByFieldBody("field1", "hello"):                              `{"text":"hello","field":"field1"}`,
		analyzeByAnalyzerBody("standard", "hello"):                         `{"text":"hello","analyzer":"standard"}`,
		updateAliasesBody(json.RawMessage(`[{"add":{}}]`)):                 `{"actions":[{"add":{}}]}`,
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("body = %s, want %s", got, want)
		}
	}
}
