package main

import (
	"fmt"
	"os"

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
		},
		Action: run,
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	cfg, err := config.NewConfig(c.String("config"))
	if err != nil {
		return err
	}

	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

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

	hosts, err := config.NewHosts(cfg.Hosts)
	if err != nil {
		return err
	}

	// Services, wired inside-out. esClient is passed for every narrow
	// capability interface since *elastic.MultiClient implements the full
	// domain.ElasticClient.
	h := &handler.Handlers{
		Auth:   authSvc,
		Hosts:  hosts,
		Log:    log,
		Secret: cfg.Secret,

		Connect:         service.NewConnectService(esClient),
		Overview:        service.NewOverviewService(esClient, esClient, esClient),
		Nodes:           service.NewNodesService(esClient, esClient),
		Navbar:          service.NewNavbarService(esClient),
		Rest:            service.NewRestService(esClient, esClient, historyRepo, cfg.RestHistorySize),
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
	}

	srv := server.NewServer(c.Int("port"), cfg, h)
	srv.Spin()

	return nil
}
