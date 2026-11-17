package server

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"

	"cerebro/internal/auth"
	"cerebro/internal/config"
	"cerebro/internal/handler"
)

// NewServer builds the Hertz engine. Route scoping for auth mirrors the
// original Scala cerebro: the login page, the auth endpoints, the hosts
// list, static assets and the favicon are reachable without a session; every
// other route is session-protected when auth is enabled.
//
// Hertz applies middleware registered via Use() only to routes registered
// AFTER the call, so the public routes are registered first, then the auth
// middleware is installed, then the protected routes. With auth disabled the
// middleware is a no-op and behavior is identical to pre-auth wiring.
func NewServer(port int, cfg *config.Config, h *handler.Handlers) *server.Hertz {
	addr := fmt.Sprintf(":%d", port)
	srv := server.Default(
		server.WithHostPorts(addr),
		server.WithBasePath(cfg.BasePath),
	)

	// --- Public routes (no session required) ---
	// The SPA shell served on /login renders the login form; /connect/hosts
	// feeds the connect screen; assets/favicon are needed to render them.
	srv.GET("/login", h.AuthIndex)
	srv.GET("/connect/hosts", h.ConnectIndex)
	srv.POST("/auth/login", h.AuthLogin)
	srv.POST("/auth/logout", h.AuthLogout)
	srv.GET("/auth/me", h.AuthMe)

	publicDir := filepath.Join("output", "public")
	// Hertz maps Root + the full request path onto the filesystem, so the
	// roots below are the parents of their URL prefixes. Compress stays
	// disabled (the FS zero value): transparent gzip caches its output next
	// to the served file, which the read-only container root filesystem
	// forbids.
	//
	// /public/* is the original cerebro asset URL space, mapping onto
	// output/public/*; /_nuxt/* serves the Nuxt build assets that the SPA
	// shell references.
	srv.StaticFS("/public", &app.FS{Root: filepath.Join("output")})
	srv.StaticFS("/_nuxt", &app.FS{Root: publicDir})
	srv.GET("/favicon.ico", func(ctx context.Context, c *app.RequestContext) {
		c.Status(http.StatusNoContent)
	})

	// --- Auth middleware: everything registered below is protected ---
	// API calls without a session get the {"status":303,"body":null} envelope
	// (HTTP 200); browser navigations get a 302 to /login. GET / and the SPA
	// fallback are protected like the original Application.index.
	srv.Use(auth.AuthMiddleware(h.Auth, cfg.Secret))

	srv.GET("/", h.ApplicationIndex)
	srv.POST("/connect", h.ConnectConnect)

	srv.POST("/rest", h.RestIndex)
	srv.POST("/rest/request", h.RestRequest)
	srv.POST("/rest/history", h.RestHistory)

	srv.POST("/overview", h.OverviewIndex)
	srv.POST("/overview/disable_shard_allocation", h.OverviewDisableShardAllocation)
	srv.POST("/overview/enable_shard_allocation", h.OverviewEnableShardAllocation)
	srv.POST("/overview/close_indices", h.OverviewCloseIndices)
	srv.POST("/overview/open_indices", h.OverviewOpenIndices)
	srv.POST("/overview/force_merge", h.OverviewForceMerge)
	srv.POST("/overview/clear_indices_cache", h.OverviewClearIndexCache)
	srv.POST("/overview/refresh_indices", h.OverviewRefreshIndex)
	srv.POST("/overview/flush_indices", h.OverviewFlushIndex)
	srv.POST("/overview/delete_indices", h.OverviewDeleteIndex)
	srv.POST("/overview/get_shard_stats", h.OverviewGetShardStats)
	srv.POST("/overview/relocate_shard", h.OverviewRelocateShard)

	srv.POST("/nodes", h.NodesIndex)
	srv.POST("/navbar", h.NavbarIndex)

	srv.POST("/analysis/indices", h.AnalysisGetIndices)
	srv.POST("/analysis/analyzers", h.AnalysisGetIndexAnalyzers)
	srv.POST("/analysis/fields", h.AnalysisGetIndexFields)
	srv.POST("/analysis/analyze/analyzer", h.AnalysisAnalyzeByAnalyzer)
	srv.POST("/analysis/analyze/field", h.AnalysisAnalyzeByField)

	srv.POST("/cluster_changes", h.ClusterChangesGet)

	srv.POST("/commons/indices", h.CommonsIndices)
	srv.POST("/commons/nodes", h.CommonsNodes)
	srv.POST("/commons/get_index_settings", h.CommonsGetIndexSettings)
	srv.POST("/commons/get_index_mapping", h.CommonsGetIndexMappings)
	srv.POST("/commons/get_node_stats", h.CommonsGetNodeStats)
	srv.POST("/commons/get_index_stats", h.CommonsGetIndexStats)

	srv.POST("/create_index/get_index_metadata", h.CreateIndexGetIndexMetadata)
	srv.POST("/create_index/create", h.CreateIndexExecute)

	srv.POST("/templates", h.TemplatesTemplates)
	srv.POST("/templates/delete", h.TemplatesDelete)
	srv.POST("/templates/create", h.TemplatesCreate)

	srv.POST("/data_streams", h.DataStreamsList)
	srv.POST("/data_streams/create", h.DataStreamsCreate)
	srv.POST("/data_streams/delete", h.DataStreamsDelete)
	srv.POST("/data_streams/rollover", h.DataStreamsRollover)

	srv.POST("/cluster_settings", h.ClusterSettingsGet)
	srv.POST("/cluster_settings/save", h.ClusterSettingsSave)

	srv.POST("/index_settings", h.IndexSettingsGet)
	srv.POST("/index_settings/update", h.IndexSettingsUpdate)

	srv.POST("/repositories", h.RepositoriesGet)
	srv.POST("/repositories/create", h.RepositoriesSave)
	srv.POST("/repositories/delete", h.RepositoriesDelete)
	srv.POST("/repositories/verify", h.RepositoriesVerify)

	srv.POST("/snapshots", h.SnapshotsGet)
	srv.POST("/snapshots/load", h.SnapshotsLoad)
	srv.POST("/snapshots/create", h.SnapshotsCreate)
	srv.POST("/snapshots/delete", h.SnapshotsDelete)
	srv.POST("/snapshots/restore", h.SnapshotsRestore)

	srv.POST("/policies/index_lifecycle", h.PoliciesIndexLifecycle)
	srv.POST("/policies/retry_indices", h.PoliciesRetryIndices)
	srv.POST("/policies/toggle_engine", h.PoliciesToggleEngine)
	srv.POST("/policies/snapshot_lifecycle", h.PoliciesSnapshotLifecycle)
	srv.POST("/policies/snapshot_action", h.PoliciesSnapshotAction)

	srv.POST("/cat", h.CatGet)

	srv.POST("/aliases/get_aliases", h.AliasesGetAliases)
	srv.POST("/aliases/update_aliases", h.AliasesUpdateAliases)

	// SPA history-mode fallback: registered after the middleware so deep links
	// also require a session when auth is enabled.
	srv.NoRoute(func(ctx context.Context, c *app.RequestContext) {
		// Uncompressed: ServeFile's gzip caching needs write access next to
		// the file, which the read-only container root filesystem forbids.
		app.ServeFileUncompressed(c, filepath.Join(publicDir, "index.html"))
	})

	return srv
}
