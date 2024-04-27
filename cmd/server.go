package cmd

import (
	"broadcaster/controllers/restapi"
	"broadcaster/services/housekeeper"
	"broadcaster/services/processer"
	"broadcaster/storages/memory"
	"broadcaster/utils/info"
	"broadcaster/utils/logging"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

type serverConfig struct {
	Env           string         `envconfig:"ENV" default:"local"`
	LogLevel      string         `envconfig:"LOG_LEVEL" default:"info"`
	LogFormat     logging.Format `envconfig:"LOG_FORMAT" default:"pretty_color"`
	BootstrapFile string         `envconfig:"BOOTSTRAP_FILE"`
	StateTTL      int            `envconfig:"STATE_TTL" default:"86400"`
	CheckInterval int            `envconfig:"CHECK_INTERVAL" default:"300"`
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs the server",
	Run: func(cmd *cobra.Command, args []string) {
		_ = godotenv.Load() // Try to read .env file first

		/* Load configuration */

		var cfg serverConfig
		if err := envconfig.Process(info.EnvPrefix, &cfg); err != nil {
			fmt.Printf("Can't load env: %s\n", err.Error())
			os.Exit(1)
		}

		/* Logger */

		logger := logging.NewLogger(cfg.LogLevel, cfg.LogFormat)
		logger.WithOptions(zap.AddStacktrace(zap.ErrorLevel))

		// Adding env fields for non-local environments
		if cfg.Env != "local" {
			logger = logger.With(
				"app", info.Namespace,
				"env", cfg.Env,
				"release", info.Release,
				"hash", info.CommitHash,
			)
		}

		/* Root context */

		ctx, cancel := context.WithCancel(context.Background())
		ctx = logging.ContextWithLogger(ctx, logger)
		defer cancel()

		// Listening for OS interupt signals
		terminateCh := make(chan os.Signal, 1)
		signal.Notify(terminateCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			select {
			case <-terminateCh:
				cancel()
			case <-ctx.Done():
			}
		}()
		defer func() {
			if r := recover(); r != nil {
				logger.Fatalw("Application panic", "panic", r)
			}
		}()

		logger.Infof("Starting %s (%s)", info.AppName, info.Release)
		defer logger.Info("App is stopped")

		/* Storage */

		st := memory.NewStorage(
			memory.WithLogger(logger.Named("storage")),
		)

		logger.Debugf("Bootstraping from config file: '%s'", cfg.BootstrapFile)

		if err := st.BootstrapFromConfigFile(ctx, cfg.BootstrapFile); err != nil {
			logger.Fatalf("Bootstrap failed: %s", err.Error())
		}

		/* Services */

		pcr, err := processer.NewService(
			st,
			processer.WithLogger(logger.Named("processer")),
		)
		if err != nil {
			logger.Fatalf("Can't create processer service: %v", err.Error())
		}

		hkr, err := housekeeper.NewService(
			st,
			housekeeper.WithLogger(logger.Named("housekeeper")),
		)
		if err != nil {
			logger.Fatalf("Can't create housekeeper service: %w", err)
		}

		/* Starting service */

		// Processing data for the first time and start regular processing
		if err := pcr.Process(ctx); err != nil {
			logger.Error("Failed to process data: ", err.Error())
		}
		go func() {
			interval := time.Duration(cfg.CheckInterval) * time.Second
			ticker := time.NewTicker(interval)
			for {
				select {
				case <-ticker.C:
					if err := pcr.Process(ctx); err != nil {
						logger.Error("Failed to process data: ", err.Error())
					}

					ttl := time.Duration(cfg.StateTTL) * time.Second

					if err := hkr.CleanupFeedItems(ctx, ttl); err != nil {
						logger.Error("Failed to cleanup feed items: ", err.Error())
					}
				case <-ctx.Done():
					logger.Info("Stopping application")
					ticker.Stop()
					return
				}
			}
		}()

		api, err := restapi.New(
			restapi.WithLogger(logger.Named("restapi")),
		)
		if err != nil {
			logger.Fatalf("Can't create restapi: %w", err)
		}

		err = api.Serve(ctx)
		cancel()
		if err != nil {
			logger.Fatal(err)
		}
	},
}
