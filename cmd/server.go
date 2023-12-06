package cmd

import (
	"a0feed/cmd/config"
	"a0feed/controllers/restapi"
	"a0feed/service"
	"a0feed/utils/info"
	"a0feed/utils/logging"
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

type serverConfig struct {
	Env           string         `envconfig:"ENV" default:"local"`
	LogLevel      string         `envconfig:"LOG_LEVEL" default:"info"`
	LogFormat     logging.Format `envconfig:"LOG_FORMAT" default:"json"`
	ConfigPath    string         `envconfig:"CONFIG_PATH"`
	CheckInterval int            `envconfig:"CHECK_INTERVAL" default:"300"`
}

func loadServerConfig() (*serverConfig, error) {
	_ = godotenv.Load() // Try to read .env file first

	var cfg serverConfig
	if err := envconfig.Process(info.EnvPrefix, &cfg); err != nil {
		return nil, fmt.Errorf("Can't load environment variables: %w", err)
	}

	// Set color logs for the local development
	if cfg.Env == "local" && cfg.LogFormat != "pretty" {
		cfg.LogFormat = "pretty_color"
	}

	return &cfg, nil
}

func init() {
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs the server",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadServerConfig()
		if err != nil {
			fmt.Printf("Failed to load configuration: %s\n", err.Error())
			os.Exit(1)
		}

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

		// Root context
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

		// Loading feeds configuration
		logger.Debugf("Loading feeds confguration from '%s'", cfg.ConfigPath)

		cdata, err := config.Load(ctx, cfg.ConfigPath)
		if err != nil {
			logger.Fatal("Failed to load config file: ", err.Error())
		}
		logger.Debugf("Loaded '%d' feeds configurations", len(cdata.Feeds))

		// Service
		svc, err := service.New(
			service.WithFeeds(cdata.Feeds...),
			service.WithLogger(logger),
		)
		if err != nil {
			logger.Fatal("Failed to load translator service: ", err.Error())
		}

		if err := svc.Process(ctx); err != nil {
			logger.Error("Failed to process data: ", err.Error())
		}

		// Starting service
		go func() {
			interval := time.Duration(cfg.CheckInterval) * time.Second
			ticker := time.NewTicker(interval)
			for {
				select {
				case <-ticker.C:
					if err := svc.Process(ctx); err != nil {
						logger.Error("Failed to process data: ", err.Error())
					}
				case <-ctx.Done():
					logger.Info("Stopping application")
					ticker.Stop()
					return
				}
			}
		}()

		api, err := restapi.New(
			restapi.WithLogger(logger),
		)
		if err != nil {
			logger.Fatal(err)
		}

		err = api.Serve(ctx)
		cancel()
		if err != nil {
			logger.Fatal(err)
		}
	},
}
