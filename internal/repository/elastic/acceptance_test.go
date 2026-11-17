//go:build acceptance

package elastic_test

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"cerebro/internal/domain"
	"cerebro/internal/repository/elastic"
	"cerebro/internal/service"
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
	target := domain.ElasticServer{Host: domain.Host{Name: url}}
	ctx := context.Background()

	// Make the suite idempotent: Dagger may reuse a cached service filesystem
	// across runs, so remove any leftovers from a previous (possibly failed) run.
	_, _ = client.DeleteIndex("cerebro-acceptance", ctx, target)
	_, _ = client.DeleteIndex("restored-cerebro-acceptance", ctx, target)
	_, _ = client.DeleteRepository("cerebro-acceptance-repo", ctx, target)
	_, _ = client.DeleteDataStream("cerebro-acceptance-ds", ctx, target)
	_, _ = client.DeleteComposableTemplate("cerebro-acceptance-ds-ds-template", ctx, target)
	_, _ = client.DeleteComposableTemplate("cerebro-acceptance-template", ctx, target)
	_, _ = client.DeleteTemplate("cerebro-acceptance-legacy", ctx, target)

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

	t.Run("Index_actions_lifecycle", func(t *testing.T) {
		actions := []struct {
			name string
			call func() (*domain.Response, error)
		}{
			{"RefreshIndex", func() (*domain.Response, error) { return client.RefreshIndex("cerebro-acceptance", ctx, target) }},
			{"FlushIndex", func() (*domain.Response, error) { return client.FlushIndex("cerebro-acceptance", ctx, target) }},
			{"ClearIndexCache", func() (*domain.Response, error) { return client.ClearIndexCache("cerebro-acceptance", ctx, target) }},
			{"ForceMerge", func() (*domain.Response, error) { return client.ForceMerge("cerebro-acceptance", ctx, target) }},
			{"CloseIndex", func() (*domain.Response, error) { return client.CloseIndex("cerebro-acceptance", ctx, target) }},
			{"OpenIndex", func() (*domain.Response, error) { return client.OpenIndex("cerebro-acceptance", ctx, target) }},
		}
		for _, action := range actions {
			resp, err := action.call()
			if err != nil {
				t.Fatalf("%s: %v", action.name, err)
			}
			if resp.Status != 200 {
				t.Fatalf("%s status = %d, want 200: %s", action.name, resp.Status, resp.Body)
			}
		}
	})

	t.Run("Templates_composable_and_legacy", func(t *testing.T) {
		composableName := "cerebro-acceptance-template"
		legacyName := "cerebro-acceptance-legacy"
		composableBody := json.RawMessage(`{"index_patterns":["cerebro-acceptance-*"],"priority":100,"template":{"settings":{"number_of_shards":1}}}`)
		legacyBody := json.RawMessage(`{"index_patterns":["cerebro-legacy-*"],"order":1,"settings":{}}`)

		createComposable, err := client.CreateComposableTemplate(composableName, composableBody, ctx, target)
		if err != nil {
			t.Fatalf("CreateComposableTemplate: %v", err)
		}
		if createComposable.Status != 200 {
			t.Fatalf("composable create status = %d: %s", createComposable.Status, createComposable.Body)
		}
		defer func() { _, _ = client.DeleteComposableTemplate(composableName, ctx, target) }()

		createLegacy, err := client.CreateTemplate(legacyName, legacyBody, ctx, target)
		if err != nil {
			t.Fatalf("CreateTemplate: %v", err)
		}
		if createLegacy.Status != 200 {
			t.Fatalf("legacy create status = %d: %s", createLegacy.Status, createLegacy.Body)
		}
		defer func() { _, _ = client.DeleteTemplate(legacyName, ctx, target) }()

		templates := service.NewTemplatesService(client)
		resp, err := templates.Templates(ctx, domain.CerebroRequest{Target: target})
		if err != nil {
			t.Fatalf("Templates: %v", err)
		}
		if resp.Status != 200 {
			t.Fatalf("templates status = %d, want 200: %s", resp.Status, resp.Body)
		}
		var entries []domain.TemplateEntry
		if err := json.Unmarshal(resp.Body, &entries); err != nil {
			t.Fatalf("unmarshal templates: %v", err)
		}
		found := map[string]bool{}
		for _, entry := range entries {
			found[entry.Name] = true
		}
		if !found[composableName] || !found[legacyName] {
			t.Errorf("templates = %v, want both %s and %s", found, composableName, legacyName)
		}
	})

	t.Run("DataStreams_create_overview_rollover_delete", func(t *testing.T) {
		dsName := "cerebro-acceptance-ds"
		templateName := dsName + "-ds-template"
		dsSvc := service.NewDataStreamsService(client, client)

		createReq := domain.CerebroRequest{
			Target: target,
			Body:   json.RawMessage(`{"name":"` + dsName + `","createTemplate":true}`),
		}
		createResp, err := dsSvc.Create(ctx, createReq)
		if err != nil {
			t.Fatalf("DataStreams.Create: %v", err)
		}
		if createResp.Status != 200 {
			t.Fatalf("create data stream status = %d: %s", createResp.Status, createResp.Body)
		}
		defer func() {
			_, _ = client.DeleteDataStream(dsName, ctx, target)
			_, _ = client.DeleteComposableTemplate(templateName, ctx, target)
		}()

		doc := json.RawMessage(`{"@timestamp":"2024-01-01T00:00:00Z","message":"hello"}`)
		indexResp, err := client.ExecuteRequest("POST", "/"+dsName+"/_doc", doc, ctx, target)
		if err != nil {
			t.Fatalf("index doc: %v", err)
		}
		if indexResp.Status != 200 && indexResp.Status != 201 {
			t.Fatalf("index doc status = %d: %s", indexResp.Status, indexResp.Body)
		}
		if _, err := client.RefreshIndex(dsName, ctx, target); err != nil {
			t.Fatalf("RefreshIndex data stream: %v", err)
		}

		overview := service.NewOverviewService(client, client, client, client, client)
		overviewResp, err := overview.Index(ctx, domain.CerebroRequest{Target: target})
		if err != nil {
			t.Fatalf("Overview.Index: %v", err)
		}
		if overviewResp.Status != 200 {
			t.Fatalf("overview status = %d: %s", overviewResp.Status, overviewResp.Body)
		}
		var body struct {
			DataStreams []domain.DataStreamSummary `json:"data_streams"`
			Indices     []domain.OverviewIndex     `json:"indices"`
		}
		if err := json.Unmarshal(overviewResp.Body, &body); err != nil {
			t.Fatalf("unmarshal overview: %v", err)
		}
		foundStream := false
		for _, ds := range body.DataStreams {
			if ds.Name == dsName {
				foundStream = true
			}
		}
		if !foundStream {
			t.Errorf("data_streams = %+v, want %s", body.DataStreams, dsName)
		}
		foundBacking := false
		for _, idx := range body.Indices {
			if idx.DataStream == dsName {
				foundBacking = true
				if !idx.BackingIndex || !idx.IsWriteIndex || idx.Special {
					t.Errorf("backing index = %+v, want backing/write/non-special", idx)
				}
			}
		}
		if !foundBacking {
			t.Errorf("no backing index for data stream %s in overview", dsName)
		}

		rolloverResp, err := dsSvc.Rollover(ctx, domain.CerebroRequest{Target: target, Body: json.RawMessage(`{"name":"` + dsName + `"}`)})
		if err != nil {
			t.Fatalf("DataStreams.Rollover: %v", err)
		}
		if rolloverResp.Status != 200 {
			t.Fatalf("rollover status = %d: %s", rolloverResp.Status, rolloverResp.Body)
		}

		deleteResp, err := dsSvc.Delete(ctx, domain.CerebroRequest{Target: target, Body: json.RawMessage(`{"name":"` + dsName + `"}`)})
		if err != nil {
			t.Fatalf("DataStreams.Delete: %v", err)
		}
		if deleteResp.Status != 200 {
			t.Fatalf("delete status = %d: %s", deleteResp.Status, deleteResp.Body)
		}
	})

	t.Run("Overview_docs_size_and_master", func(t *testing.T) {
		// The bulk doc must be refreshed before _stats reports it.
		if _, err := client.RefreshIndex("cerebro-acceptance", ctx, target); err != nil {
			t.Fatalf("RefreshIndex: %v", err)
		}
		overview := service.NewOverviewService(client, client, client, client, client)
		resp, err := overview.Index(ctx, domain.CerebroRequest{Target: target})
		if err != nil {
			t.Fatalf("Overview.Index: %v", err)
		}
		if resp.Status != 200 {
			t.Fatalf("status = %d, want 200: %s", resp.Status, resp.Body)
		}
		var body struct {
			DocsCount   int64 `json:"docs_count"`
			SizeInBytes int64 `json:"size_in_bytes"`
			Nodes       []struct {
				CurrentMaster bool `json:"current_master"`
			} `json:"nodes"`
		}
		if err := json.Unmarshal(resp.Body, &body); err != nil {
			t.Fatalf("unmarshal overview: %v", err)
		}
		if body.DocsCount < 1 {
			t.Errorf("docs_count = %d, want >= 1", body.DocsCount)
		}
		if body.SizeInBytes <= 0 {
			t.Errorf("size_in_bytes = %d, want > 0", body.SizeInBytes)
		}
		masters := 0
		for _, n := range body.Nodes {
			if n.CurrentMaster {
				masters++
			}
		}
		if masters != 1 {
			t.Errorf("current masters = %d, want exactly 1", masters)
		}
	})

	t.Run("Nodes_cpu_load_and_master", func(t *testing.T) {
		nodes := service.NewNodesService(client, client)
		resp, err := nodes.Index(ctx, domain.CerebroRequest{Target: target})
		if err != nil {
			t.Fatalf("Nodes.Index: %v", err)
		}
		if resp.Status != 200 {
			t.Fatalf("status = %d, want 200: %s", resp.Status, resp.Body)
		}
		var entries []domain.NodePageEntry
		if err := json.Unmarshal(resp.Body, &entries); err != nil {
			t.Fatalf("unmarshal nodes: %v", err)
		}
		if len(entries) == 0 {
			t.Fatal("no nodes returned")
		}
		masters := 0
		for _, n := range entries {
			if n.CurrentMaster {
				masters++
			}
			if n.CPUPercent != nil && *n.CPUPercent < 0 {
				t.Errorf("node %s cpu_percent = %v, want non-negative", n.Name, *n.CPUPercent)
			}
			if n.OSCPUPercent != nil && *n.OSCPUPercent < 0 {
				t.Errorf("node %s os_cpu_percent = %d, want non-negative", n.Name, *n.OSCPUPercent)
			}
			if n.ProcessCPUPercent != nil && *n.ProcessCPUPercent < 0 {
				t.Errorf("node %s process_cpu_percent = %d, want non-negative", n.Name, *n.ProcessCPUPercent)
			}
			if n.ResourceCPUPercent != nil && *n.ResourceCPUPercent < 0 {
				t.Errorf("node %s resource_cpu_percent = %v, want non-negative", n.Name, *n.ResourceCPUPercent)
			}
		}
		if masters != 1 {
			t.Errorf("current masters = %d, want exactly 1", masters)
		}

		// /overview must expose the same non-negative, source-aware CPU data.
		overview := service.NewOverviewService(client, client, client, client, client)
		overviewResp, err := overview.Index(ctx, domain.CerebroRequest{Target: target})
		if err != nil {
			t.Fatalf("Overview.Index: %v", err)
		}
		if overviewResp.Status != 200 {
			t.Fatalf("overview status = %d, want 200: %s", overviewResp.Status, overviewResp.Body)
		}
		var overviewBody struct {
			Nodes []struct {
				Name       string   `json:"name"`
				CPUPercent *float64 `json:"cpu_percent"`
			} `json:"nodes"`
		}
		if err := json.Unmarshal(overviewResp.Body, &overviewBody); err != nil {
			t.Fatalf("unmarshal overview: %v", err)
		}
		for _, n := range overviewBody.Nodes {
			if n.CPUPercent != nil && *n.CPUPercent < 0 {
				t.Errorf("overview node %s cpu_percent = %v, want non-negative", n.Name, *n.CPUPercent)
			}
		}

		resourceResp, err := client.NodesResourceUsageStats(ctx, target)
		if err != nil {
			t.Fatalf("NodesResourceUsageStats: %v", err)
		}
		switch wantFlavor {
		case elastic.FlavorElasticsearch:
			if resourceResp.Status != 404 {
				t.Errorf("elasticsearch resource_usage_stats status = %d, want synthetic 404", resourceResp.Status)
			}
		case elastic.FlavorOpenSearch:
			if resourceResp.Status != 200 {
				t.Logf("OpenSearch resource_usage_stats unavailable (%d); CPU gracefully falls back", resourceResp.Status)
				break
			}
			var resourceBody struct {
				Nodes map[string]struct {
					ResourceUsageStats struct {
						CPUUtilizationPercent *float64 `json:"cpu_utilization_percent"`
					} `json:"resource_usage_stats"`
				} `json:"nodes"`
			}
			if err := json.Unmarshal(resourceResp.Body, &resourceBody); err != nil {
				t.Fatalf("unmarshal resource_usage_stats: %v", err)
			}
			byID := make(map[string]domain.NodePageEntry, len(entries))
			for _, n := range entries {
				byID[n.ID] = n
			}
			for id, node := range resourceBody.Nodes {
				cpu := node.ResourceUsageStats.CPUUtilizationPercent
				if cpu == nil {
					continue
				}
				entry, ok := byID[id]
				if !ok {
					continue
				}
				if entry.CPUSource != domain.CPUSourceResourceTracker {
					t.Errorf("node %s cpu_source = %q, want %q", id, entry.CPUSource, domain.CPUSourceResourceTracker)
				}
				if want := math.Round(*cpu*100) / 100; entry.CPUPercent == nil || *entry.CPUPercent != want {
					t.Errorf("node %s cpu_percent = %v, want %v", id, entry.CPUPercent, want)
				}
			}
		}
	})

	t.Run("ClusterSettings_persistent_roundtrip", func(t *testing.T) {
		settings := service.NewClusterSettingsService(client)
		save := domain.CerebroRequest{
			Target: target,
			Body:   json.RawMessage(`{"settings":{"persistent":{"cluster.routing.allocation.disk.threshold_enabled":true}}}`),
		}
		resp, err := settings.Save(ctx, save)
		if err != nil {
			t.Fatalf("ClusterSettings.Save: %v", err)
		}
		if resp.Status != 200 {
			t.Fatalf("save status = %d, want 200: %s", resp.Status, resp.Body)
		}
		get, err := settings.Get(ctx, domain.CerebroRequest{Target: target})
		if err != nil {
			t.Fatalf("ClusterSettings.Get: %v", err)
		}
		if !strings.Contains(string(get.Body), "cluster.routing.allocation.disk.threshold_enabled") {
			t.Errorf("settings body missing saved key: %s", get.Body)
		}
		reset := domain.CerebroRequest{
			Target: target,
			Body:   json.RawMessage(`{"settings":{"persistent":{"cluster.routing.allocation.disk.threshold_enabled":null}}}`),
		}
		if resp, err := settings.Save(ctx, reset); err != nil || resp.Status != 200 {
			t.Fatalf("reset status = %d, err = %v: %s", resp.Status, err, resp.Body)
		}
	})

	t.Run("Repository_create_verify_delete", func(t *testing.T) {
		repoName := "cerebro-acceptance-repo"
		create, err := client.CreateRepository(repoName, "fs", json.RawMessage(`{"location":"/tmp/cerebro-acceptance-repo","compress":true}`), ctx, target)
		if err != nil {
			t.Fatalf("CreateRepository: %v", err)
		}
		if create.Status != 200 {
			t.Fatalf("create status = %d: %s", create.Status, create.Body)
		}
		defer func() {
			_, _ = client.DeleteRepository(repoName, ctx, target)
		}()

		repos := service.NewRepositoriesService(client)
		resp, err := repos.Verify(ctx, domain.CerebroRequest{
			Target: target,
			Body:   json.RawMessage(`{"name":"` + repoName + `"}`),
		})
		if err != nil {
			t.Fatalf("Repositories.Verify: %v", err)
		}
		if resp.Status != 200 {
			t.Fatalf("verify status = %d: %s", resp.Status, resp.Body)
		}
		var status domain.RepositoryVerifyStatus
		if err := json.Unmarshal(resp.Body, &status); err != nil {
			t.Fatalf("unmarshal verify: %v", err)
		}
		if !status.Verified || status.NodeCount == 0 {
			t.Errorf("verify = %+v, want verified with nodes", status)
		}
	})

	t.Run("Snapshot_create_restore", func(t *testing.T) {
		repoName := "cerebro-acceptance-repo"
		if _, err := client.CreateRepository(repoName, "fs", json.RawMessage(`{"location":"/tmp/cerebro-acceptance-repo","compress":true}`), ctx, target); err != nil {
			t.Fatalf("CreateRepository: %v", err)
		}
		defer func() {
			_, _ = client.DeleteRepository(repoName, ctx, target)
		}()

		snapName := "cerebro-acceptance-snap"
		create, err := client.CreateSnapshot(repoName, snapName, true, false, "cerebro-acceptance", ctx, target)
		if err != nil {
			t.Fatalf("CreateSnapshot: %v", err)
		}
		if create.Status != 200 {
			t.Fatalf("create snapshot status = %d: %s", create.Status, create.Body)
		}
		waitForSnapshot(t, client, repoName, snapName, ctx, target)

		repos := service.NewRepositoriesService(client)
		verifyResp, err := repos.Verify(ctx, domain.CerebroRequest{
			Target: target,
			Body:   json.RawMessage(`{"name":"` + repoName + `"}`),
		})
		if err != nil {
			t.Fatalf("Repositories.Verify after snapshot: %v", err)
		}
		if verifyResp.Status != 200 {
			t.Fatalf("verify status = %d: %s", verifyResp.Status, verifyResp.Body)
		}
		var verifyStatus domain.RepositoryVerifyStatus
		if err := json.Unmarshal(verifyResp.Body, &verifyStatus); err != nil {
			t.Fatalf("unmarshal verify: %v", err)
		}
		if !verifyStatus.SizeComputed || verifyStatus.SizeInBytes == nil || *verifyStatus.SizeInBytes <= 0 {
			t.Errorf("verify size = %+v, want computed non-zero size", verifyStatus)
		}

		restore, err := client.RestoreSnapshot(repoName, snapName, domain.RestoreSnapshotOptions{
			Indices:           "cerebro-acceptance",
			RenamePattern:     "(.+)",
			RenameReplacement: "restored-$1",
			IgnoreUnavailable: true,
			IncludeAliases:    true,
		}, ctx, target)
		if err != nil {
			t.Fatalf("RestoreSnapshot: %v", err)
		}
		if restore.Status != 200 {
			t.Fatalf("restore status = %d: %s", restore.Status, restore.Body)
		}
		defer func() {
			_, _ = client.DeleteIndex("restored-cerebro-acceptance", ctx, target)
		}()
		waitForIndex(t, client, "restored-cerebro-acceptance", ctx, target)
	})

	t.Run("Policies_list", func(t *testing.T) {
		policies := service.NewPoliciesService(client, client)
		indexResp, err := policies.ListIndexPolicies(ctx, domain.CerebroRequest{Target: target})
		if err != nil {
			t.Fatalf("ListIndexPolicies: %v", err)
		}
		if indexResp.Status != 200 {
			t.Fatalf("index policies status = %d: %s", indexResp.Status, indexResp.Body)
		}
		snapResp, err := policies.ListSnapshotPolicies(ctx, domain.CerebroRequest{Target: target})
		if err != nil {
			t.Fatalf("ListSnapshotPolicies: %v", err)
		}
		if snapResp.Status != 200 {
			t.Fatalf("snapshot policies status = %d: %s", snapResp.Status, snapResp.Body)
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

func waitForSnapshot(t *testing.T, client domain.ElasticClient, repo, snap string, ctx context.Context, target domain.ElasticServer) {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.GetSnapshots(repo, ctx, target)
		if err == nil && resp.IsSuccess() {
			var body struct {
				Snapshots []struct {
					Snapshot string `json:"snapshot"`
					State    string `json:"state"`
				} `json:"snapshots"`
			}
			_ = json.Unmarshal(resp.Body, &body)
			for _, s := range body.Snapshots {
				if s.Snapshot == snap && s.State == "SUCCESS" {
					return
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("snapshot %s did not reach SUCCESS", snap)
}

func waitForIndex(t *testing.T, client domain.ElasticClient, index string, ctx context.Context, target domain.ElasticServer) {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.GetIndices(ctx, target)
		if err == nil && resp.IsSuccess() && strings.Contains(string(resp.Body), index) {
			return
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("index %s did not appear after restore", index)
}
