package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"cerebro/internal/auth"
	"cerebro/internal/config"
	"cerebro/internal/repository"
	"cerebro/internal/repository/elastic"
	"cerebro/internal/handlers"
	"cerebro/internal/server"
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

	var hist *repository.SQLiteHistoryRepository
	if cfg.DataPath != "" {
		hist, err = repository.NewSQLiteHistoryRepository(cfg.DataPath)
		if err != nil {
			return err
		}
		defer func() { _ = hist.Close() }()
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

	h := &handlers.Handlers{
		Auth:    authSvc,
		Hosts:   hosts,
		Elastic: esClient,
		History: hist,
		HistMax: cfg.RestHistorySize,
		Log:     log,
		Secret:  cfg.Secret,
	}

	srv := server.NewServer(c.Int("port"), cfg, h)
	srv.Spin()

	return nil
}
