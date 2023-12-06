package cmd

import (
	"a0feed/cmd/config"
	"a0feed/service"
	"a0feed/utils/info"
	"a0feed/utils/logging"
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func init() {
	rootCmd.AddCommand(jobCmd)
}

var jobCmd = &cobra.Command{
	Use:   "job",
	Short: "One-term job run",
	Run: func(cmd *cobra.Command, args []string) {
		_ = godotenv.Load() // Try to read .env file first

		var cfg struct {
			Env        string `envconfig:"ENV" default:"local"`
			LogLevel   string `envconfig:"LOG_LEVEL" default:"info"`
			DevMode    bool   `envconfig:"DEV_MODE"`
			ConfigPath string `envconfig:"CONFIG_PATH"`
		}
		if err := envconfig.Process(info.EnvPrefix, &cfg); err != nil {
			fmt.Printf("Failed to load configuration: %s\n", err.Error())
			os.Exit(1)
		}
		if cfg.Env == "local" && !cfg.DevMode {
			cfg.DevMode = true
			_ = os.Setenv(info.EnvPrefix+"_DEV_MODE", "true")
		}
		logger := logging.NewLogger(cfg.LogLevel, cfg.DevMode)
		if !cfg.DevMode {
			logger = logger.With(
				"app", info.Namespace,
				"env", cfg.Env,
				"release", info.Release,
				"hash", info.CommitHash,
			)
		}
		logger.WithOptions(zap.AddStacktrace(zap.ErrorLevel))

		// Root context
		ctx, cancel := context.WithCancel(context.Background())
		ctx = logging.ContextWithLogger(ctx, logger)
		defer cancel()

		logger.Infof("Starting %s %s", info.AppName, info.Release)
		defer logger.Info("App is stopped")

		// Loading feeds configuration
		logger.Infof("Loading feeds confguration from '%s'", cfg.ConfigPath)
		cdata, err := config.Load(ctx, cfg.ConfigPath)
		if err != nil {
			logger.Fatal("Failed to load config file: ", err.Error())
		}
		logger.Infof("Loaded '%d' feeds configurations", len(cdata.Feeds))

		// Service
		svc, err := service.New(
			service.WithFeeds(cdata.Feeds...),
			service.WithLogger(logger),
		)
		if err != nil {
			logger.Fatal("Failed to load translator service: ", err.Error())
		}

		if err := svc.Process(ctx); err != nil {
			logger.Fatal("Failed to process data: ", err.Error())
		}
	},
}
