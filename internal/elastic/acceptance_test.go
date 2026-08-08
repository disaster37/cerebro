//go:build acceptance

package elastic_test

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"cerebro/internal/elastic"
	"cerebro/internal/models"
	"cerebro/internal/testutil"
)

// TestMain polls BOTH endpoints (180s budget — OpenSearch cold start is slow)
// before running the acceptance suite. Dagger must bind services "es" and
// "opensearch" and set ELASTICSEARCH_URL / OPENSEARCH_URL.
func TestMain(m *testing.M) {
	ctx := context.Background()
	if err := testutil.WaitForEndpoint(ctx, testutil.ESURL(), 180*time.Second); err != nil {
		log.Fatalf("Elasticsearch endpoint not ready: %v "+
			"(Dagger must bind the 'es' service and set ELASTICSEARCH_URL)", err)
	}
	if err := testutil.WaitForEndpoint(ctx, testutil.OpenSearchURL(), 180*time.Second); err != nil {
		log.Fatalf("OpenSearch endpoint not ready: %v "+
			"(Dagger must bind the 'opensearch' service and set OPENSEARCH_URL)", err)
	}
	os.Exit(m.Run())
}

// TestDualBackend runs the SAME suite against both real clusters.
func TestDualBackend(t *testing.T) {
	cases := []struct {
		name       string
		url        string
		want       elastic.Flavor
		wantVerPfx string
	}{
		{"elasticsearch", testutil.ESURL(), elastic.FlavorElasticsearch, "9."},
		{"opensearch", testutil.OpenSearchURL(), elastic.FlavorOpenSearch, "3."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runAcceptanceSuite(t, tc.url, tc.want, tc.wantVerPfx)
		})
	}
}

func runAcceptanceSuite(t *testing.T, url string, wantFlavor elastic.Flavor, wantVerPfx string) {
	client, err := testutil.NewAcceptanceClient()
	if err != nil {
		t.Fatalf("NewAcceptanceClient: %v", err)
	}
	target := models.ElasticServer{Host: models.Host{Name: url}}
	ctx := context.Background()

	t.Run("Main_detects_flavor_and_version", func(t *testing.T) {
		resp, err := client.Main(ctx, target)
		if err != nil {
			t.Fatalf("Main: %v", err)
		}
		if resp.Status != 200 {
			t.Fatalf("Main status = %d, want 200", resp.Status)
		}
		var root map[string]json.RawMessage
		if err := json.Unmarshal(resp.Body, &root); err != nil {
			t.Fatalf("unmarshal Main body: %v", err)
		}
		verRaw, ok := root["version"]
		if !ok {
			t.Fatal("Main body missing version block")
		}
		var ver map[string]json.RawMessage
		if err := json.Unmarshal(verRaw, &ver); err != nil {
			t.Fatalf("unmarshal version: %v", err)
		}
		var num string
		if err := json.Unmarshal(ver["number"], &num); err != nil {
			t.Fatalf("unmarshal version.number: %v", err)
		}
		if !strings.HasPrefix(num, wantVerPfx) {
			t.Errorf("version.number = %q, want prefix %q", num, wantVerPfx)
		}

		bi, err := client.BackendInfo(ctx, target)
		if err != nil {
			t.Fatalf("BackendInfo: %v", err)
		}
		if bi.Type != string(wantFlavor) {
			t.Errorf("BackendInfo.Type = %q, want %q", bi.Type, wantFlavor)
		}
		if bi.Version == "" {
			t.Error("BackendInfo.Version empty, want non-empty")
		}
		if !strings.HasPrefix(bi.Version, wantVerPfx) {
			t.Errorf("BackendInfo.Version = %q, want prefix %q", bi.Version, wantVerPfx)
		}
		if bi.Version != num {
			t.Errorf("BackendInfo.Version = %q, want version.number %q", bi.Version, num)
		}
	})

	t.Run("ClusterHealth", func(t *testing.T) {
		resp, err := client.ClusterHealth(ctx, target)
		if err != nil {
			t.Fatalf("ClusterHealth: %v", err)
		}
		if resp.Status != 200 {
			t.Fatalf("status = %d, want 200", resp.Status)
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(resp.Body, &m); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		var status string
		_ = json.Unmarshal(m["status"], &status)
		if status != "green" && status != "yellow" {
			t.Errorf("status = %q, want green or yellow", status)
		}
		var clusterName string
		_ = json.Unmarshal(m["cluster_name"], &clusterName)
		if clusterName == "" {
			t.Error("cluster_name empty")
		}
	})

	t.Run("CreateIndex", func(t *testing.T) {
		resp, err := client.CreateIndex("cerebro-acceptance", json.RawMessage(`{"settings":{"number_of_shards":1,"number_of_replicas":0}}`), ctx, target)
		if err != nil {
			t.Fatalf("CreateIndex: %v", err)
		}
		if resp.Status != 200 {
			t.Fatalf("status = %d, want 200: %s", resp.Status, resp.Body)
		}
	})

	t.Run("GetIndices", func(t *testing.T) {
		resp, err := client.GetIndices(ctx, target)
		if err != nil {
			t.Fatalf("GetIndices: %v", err)
		}
		if resp.Status != 200 {
			t.Fatalf("status = %d, want 200", resp.Status)
		}
		if !strings.Contains(string(resp.Body), "cerebro-acceptance") {
			t.Errorf("body = %s, want to contain cerebro-acceptance", resp.Body)
		}
	})

	t.Run("UpdateAliases_then_GetAliases", func(t *testing.T) {
		changes := json.RawMessage(`[{"add":{"index":"cerebro-acceptance","alias":"cerebro-acceptance-alias"}}]`)
		resp, err := client.UpdateAliases(changes, ctx, target)
		if err != nil {
			t.Fatalf("UpdateAliases: %v", err)
		}
		if resp.Status != 200 {
			t.Fatalf("UpdateAliases status = %d, want 200: %s", resp.Status, resp.Body)
		}
		aliases, err := client.GetAliases(ctx, target)
		if err != nil {
			t.Fatalf("GetAliases: %v", err)
		}
		if aliases.Status != 200 {
			t.Fatalf("GetAliases status = %d, want 200", aliases.Status)
		}
		if !strings.Contains(string(aliases.Body), "cerebro-acceptance-alias") {
			t.Errorf("aliases body = %s, want to contain cerebro-acceptance-alias", aliases.Body)
		}
	})

	t.Run("ExecuteRequest_json", func(t *testing.T) {
		resp, err := client.ExecuteRequest("GET", "/_cluster/settings", nil, ctx, target)
		if err != nil {
			t.Fatalf("ExecuteRequest json: %v", err)
		}
		if resp.Status != 200 {
			t.Fatalf("status = %d, want 200: %s", resp.Status, resp.Body)
		}
	})

	t.Run("ExecuteRequest_ndjson_bulk", func(t *testing.T) {
		ndjson := `{"index":{"_index":"cerebro-acceptance","_id":"1"}}` + "\n" + `{"field":"value"}` + "\n"
		resp, err := client.ExecuteRequest("POST", "/_bulk", json.RawMessage(ndjson), ctx, target)
		if err != nil {
			t.Fatalf("ExecuteRequest ndjson: %v", err)
		}
		if resp.Status != 200 {
			t.Fatalf("status = %d, want 200: %s", resp.Status, resp.Body)
		}
		if strings.Contains(string(resp.Body), `"errors":true`) {
			t.Errorf("bulk body contains errors: %s", resp.Body)
		}
		if !strings.Contains(string(resp.Body), `"items"`) {
			t.Errorf("bulk body = %s, want to contain items", resp.Body)
		}
	})

	t.Run("ExecuteRequest_404_passthrough", func(t *testing.T) {
		resp, err := client.ExecuteRequest("GET", "/cerebro-definitely-missing", nil, ctx, target)
		if err != nil {
			t.Fatalf("ExecuteRequest 404: %v", err)
		}
		if resp.Status != 404 {
			t.Errorf("status = %d, want 404 (non-2xx passthrough)", resp.Status)
		}
	})

	t.Run("DeleteIndex", func(t *testing.T) {
		resp, err := client.DeleteIndex("cerebro-acceptance", ctx, target)
		if err != nil {
			t.Fatalf("DeleteIndex: %v", err)
		}
		if resp.Status != 200 {
			t.Fatalf("first DeleteIndex status = %d, want 200: %s", resp.Status, resp.Body)
		}
		// Second delete -> 404 (proves the engine reports real cluster errors).
		resp2, err := client.DeleteIndex("cerebro-acceptance", ctx, target)
		if err != nil {
			t.Fatalf("second DeleteIndex: %v", err)
		}
		if resp2.Status != 404 {
			t.Errorf("second DeleteIndex status = %d, want 404", resp2.Status)
		}
	})
}
