package cmd

import (
	"a0feed/cmd/config"
	"a0feed/service"
	"a0feed/utils/info"
	"a0feed/utils/logging"
	"context"
	"fmt"
	"os"

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
			logger.Fatal("Failed to process data: ", err.Error())
		}
	},
}
