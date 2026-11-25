package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"cerebro/internal/auth"
	"cerebro/internal/config"
	"cerebro/internal/domain"
	"cerebro/internal/handler"
	"cerebro/internal/repository"
	elastic "cerebro/internal/repository/elastic"
	"cerebro/internal/server"
	"cerebro/internal/service"
)

func main() {
	app := &cli.App{
		Name:  "cerebro",
		Usage: "Elasticsearch admin web tool",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "conf/application.yaml",
				Usage:   "Path to configuration file",
			},
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Value:   9000,
				Usage:   "HTTP port to listen on",
				EnvVars: []string{"CEREBRO_PORT"},
			},
			&cli.BoolFlag{
				Name:    "config-watch",
				Value:   true,
				Usage:   "Watch configuration file for changes and hot-reload reloadable settings (hosts)",
				EnvVars: []string{"CEREBRO_CONFIG_WATCH"},
			},
			&cli.DurationFlag{
				Name:    "config-watch-interval",
				Value:   config.DefaultPollInterval,
				Usage:   "Polling fallback interval for configuration file changes (0 disables polling)",
				EnvVars: []string{"CEREBRO_CONFIG_WATCH_INTERVAL"},
			},
		},
		Action: run,
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	pollInterval := c.Duration("config-watch-interval")
	if pollInterval < 0 {
		return fmt.Errorf("--config-watch-interval must be >= 0 (got %s)", pollInterval)
	}

	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	var esClientAtomic atomic.Pointer[elastic.MultiClient]

	mgr, err := config.NewManager(
		c.String("config"),
		config.WithLogger(log),
		config.WithPollInterval(pollInterval),
		config.WithOnReload(func(_, _ *config.Snapshot) {
			if client := esClientAtomic.Load(); client != nil {
				client.ClearFlavorCache()
			}
		}),
	)
	if err != nil {
		return err
	}
	defer func() { _ = mgr.Close() }()

	snap := mgr.Snapshot()
	cfg := snap.Config

	// repository: optional REST history (nil interface when data.path unset).
	// Assign only inside the branch so a nil concrete pointer is never stuffed
	// into the interface (which would make it non-nil and break the
	// s.history == nil guard in RestService).
	var historyRepo domain.RestHistoryRepository
	if cfg.DataPath != "" {
		r, err := repository.NewSQLiteHistoryRepository(cfg.DataPath)
		if err != nil {
			return err
		}
		defer func() { _ = r.Close() }()
		historyRepo = r
	}

	authSvc, err := auth.NewAuthService(cfg.Auth)
	if err != nil {
		return err
	}
	if cfg.Auth.Type != "" {
		// Fail fast on a misconfigured signing secret: with an empty secret
		// securecookie cannot encode sessions, so nobody could ever log in.
		if cfg.Secret == "" {
			return fmt.Errorf("auth is enabled (auth.type=%q) but secret is empty: set a secret to sign session cookies", cfg.Auth.Type)
		}
		if cfg.Secret == config.DefaultSecret {
			log.Warn("auth is enabled and secret is the sample value shipped in conf/application.yaml: " +
				"set a unique secret, or session cookies can be forged by anyone who knows the default")
		}
	}

	esClient, err := elastic.NewMultiClient(cfg.ESGzip)
	if err != nil {
		return err
	}
	esClientAtomic.Store(esClient)

	// Services, wired inside-out. esClient is passed for every narrow
	// capability interface since *elastic.MultiClient implements the full
	// domain.ElasticClient.
	h := &handler.Handlers{
		Auth:   authSvc,
		Hosts:  mgr,
		Log:    log,
		Secret: cfg.Secret,

		Connect:         service.NewConnectService(esClient),
		Overview:        service.NewOverviewService(esClient, esClient, esClient, esClient, esClient),
		Nodes:           service.NewNodesService(esClient, esClient),
		Navbar:          service.NewNavbarService(esClient),
		Rest:            service.NewRestService(esClient, esClient, esClient, esClient, historyRepo, cfg.RestHistorySize),
		Aliases:         service.NewAliasesService(esClient),
		Analysis:        service.NewAnalysisService(esClient, esClient),
		Cat:             service.NewCatService(esClient),
		ClusterChanges:  service.NewClusterChangesService(esClient),
		ClusterSettings: service.NewClusterSettingsService(esClient),
		Commons:         service.NewCommonsService(esClient, esClient),
		CreateIndex:     service.NewCreateIndexService(esClient),
		IndexSettings:   service.NewIndexSettingsService(esClient),
		Repositories:    service.NewRepositoriesService(esClient),
		Snapshots:       service.NewSnapshotsService(esClient, esClient),
		Templates:       service.NewTemplatesService(esClient),
		Policies:        service.NewPoliciesService(esClient, esClient),
		DataStreams:     service.NewDataStreamsService(esClient, esClient),
	}

	watchCtx, watchCancel := context.WithCancel(context.Background())
	defer watchCancel()
	if c.Bool("config-watch") {
		go mgr.Watch(watchCtx, log)
	}

	srv := server.NewServer(c.Int("port"), cfg, h)
	srv.SetCustomSignalWaiter(makeSignalWaiter(mgr, log, watchCancel))
	srv.Spin()

	return nil
}

func makeSignalWaiter(mgr *config.Manager, log *logrus.Logger, stopWatch context.CancelFunc) func(chan error) error {
	return waitForSignals(mgr, log, stopWatch, nil)
}

// waitForSignals installs Hertz's custom signal waiter. Unlike Hertz's default
// waiter, SIGHUP reloads the configuration instead of shutting the server
// down. When ready is non-nil it is closed once signal notification is
// installed (used by tests to avoid racing the first signal).
func waitForSignals(mgr *config.Manager, log *logrus.Logger, stopWatch context.CancelFunc, ready chan<- struct{}) func(chan error) error {
	return func(errCh chan error) error {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		defer signal.Stop(signals)
		if ready != nil {
			close(ready)
		}

		for {
			select {
			case sig := <-signals:
				if sig == syscall.SIGHUP {
					log.Info("SIGHUP received: triggering configuration reload")
					if changed, err := mgr.Reload(); err == nil && !changed {
						log.Info("SIGHUP reload complete: configuration unchanged")
					}
					continue
				}
				log.WithField("signal", sig.String()).Info("shutdown signal received")
				stopWatch()
				return nil
			case err := <-errCh:
				stopWatch()
				return err
			}
		}
	}
}
