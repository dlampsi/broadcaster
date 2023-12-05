package cmd

import (
	"a0feed/service"
	"a0feed/structs"
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
	"gopkg.in/yaml.v3"
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs the server",
	Run: func(cmd *cobra.Command, args []string) {
		_ = godotenv.Load() // Try to read .env file first

		var cfg struct {
			Env           string `envconfig:"ENV" default:"local"`
			LogLevel      string `envconfig:"LOG_LEVEL" default:"info"`
			DevMode       bool   `envconfig:"DEV_MODE"`
			CheckInterval int    `envconfig:"CHECK_INTERVAL" default:"300"`
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

		logger.Infof("Starting %s %s", info.AppName, info.Release)
		defer logger.Info("App is stopped")

		// Loading feeds configuration from file
		var cdata struct {
			Feeds []structs.FeedConfig `yaml:"feeds"`
		}
		cfile, err := os.ReadFile("config.yml")
		if err != nil {
			logger.Fatal("Failed to load config file: ", err.Error())
		}
		err = yaml.Unmarshal(cfile, &cdata)
		if err != nil {
			logger.Fatal("Failed to parse config file: ", err.Error())
		}
		logger.Infof("Loaded '%d' feeds configuration from file", len(cdata.Feeds))

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

		interval := time.Duration(cfg.CheckInterval) * time.Second
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := svc.Process(ctx); err != nil {
					logger.Error("Failed to process data: ", err.Error())
				}
			case <-ctx.Done():
				logger.Info("Stopping application")
				return
			}
		}
	},
}
