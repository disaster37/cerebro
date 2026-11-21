package domain

import (
	"encoding/json"
	"testing"
)

func TestValidateClusterSettingsPayload(t *testing.T) {
	flat := json.RawMessage(`{
		"defaults": {
			"cluster.routing.allocation.enable": "all",
			"cluster.max_shards_per_node": "1000",
			"node.name": "n1"
		},
		"persistent": {"cluster.routing.allocation.disk.threshold_enabled": "true"},
		"transient": {}
	}`)

	t.Run("known parameter passes", func(t *testing.T) {
		unknown, err := ValidateClusterSettingsPayload(json.RawMessage(`{"persistent":{"cluster.max_shards_per_node":"2000"}}`), flat)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if len(unknown) != 0 {
			t.Errorf("unknown = %v, want none", unknown)
		}
	})

	t.Run("logger dynamic prefix passes", func(t *testing.T) {
		unknown, err := ValidateClusterSettingsPayload(json.RawMessage(`{"transient":{"logger.org.elasticsearch.discovery":"DEBUG"}}`), flat)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if len(unknown) != 0 {
			t.Errorf("unknown = %v, want none", unknown)
		}
	})

	t.Run("typo parameter rejected", func(t *testing.T) {
		unknown, err := ValidateClusterSettingsPayload(json.RawMessage(`{"persistent":{"cluster.max_shards_per_nod":"2000"}}`), flat)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if len(unknown) != 1 || unknown[0] != "cluster.max_shards_per_nod" {
			t.Errorf("unknown = %v, want [cluster.max_shards_per_nod]", unknown)
		}
	})

	t.Run("null reset of known key passes", func(t *testing.T) {
		unknown, err := ValidateClusterSettingsPayload(json.RawMessage(`{"persistent":{"cluster.max_shards_per_node":null}}`), flat)
		if err != nil {
			t.Fatalf("Validate: %v", err)
		}
		if len(unknown) != 0 {
			t.Errorf("unknown = %v, want none", unknown)
		}
	})
}
