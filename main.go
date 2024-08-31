package main

import (
	"broadcaster/utils/info"
	"broadcaster/utils/logging"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dlampsi/gsrv"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/mmcdole/gofeed"
	"go.uber.org/zap"
)

var feeds = []feed{
	{ID: "reddit.golang", Name: "Reddit Golang", URL: "https://www.reddit.com/r/golang/.rss"},
	{ID: "reddit.programming", Name: "Reddit Programming", URL: "https://www.reddit.com/r/programming/.rss"},
	{ID: "reddit.memes", Name: "Reddit Memes", URL: "https://www.reddit.com/r/memes/.rss"},
	{ID: "reddit.helsinki", Name: "Reddit Helsinki", URL: "https://www.reddit.com/r/helsinki/.rss"},
}

func main() {
	_ = godotenv.Load() // Try to read .env file first

	var cfg struct {
		LogLevel  string         `envconfig:"LOG_LEVEL" default:"info"`
		LogFormat logging.Format `envconfig:"LOG_FORMAT" default:"pretty_color"`
	}
	if err := envconfig.Process(info.EnvPrefix, &cfg); err != nil {
		fmt.Printf("Can't load env: %s\n", err.Error())
		os.Exit(1)
	}

	logger := logging.NewLogger(cfg.LogLevel, cfg.LogFormat)
	logger.WithOptions(zap.AddStacktrace(zap.ErrorLevel))

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

	svc := &service{
		logger: logger.Named("processer"),
		feeds:  feeds,
	}

	svc.Run(ctx, 0)

	go svc.Run(ctx, time.Duration(30)*time.Second)

	srv, err := gsrv.New("0.0.0.0:8080", gsrv.WithLogger(logger))
	if err != nil {
		logger.Fatal(err)
	}
	srv.ServeHTTP(ctx, nil)
}

type feed struct {
	ID   string
	Name string
	URL  string
}

type service struct {
	logger  *zap.SugaredLogger
	lastRun *time.Time
	feeds   []feed
}

func (s *service) Run(ctx context.Context, interval time.Duration) {
	if interval == 0 {
		s.process(ctx)
		return
	}

	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			s.process(ctx)
		case <-ctx.Done():
			s.logger.Info("Stopping application")
			ticker.Stop()
			return
		}
	}
}

func (s *service) process(ctx context.Context) {
	s.logger.Debug("Started...")
	defer s.logger.Debug("Finished")

	now := time.Now()

	now = now.Add(-time.Minute * 15)

	if s.lastRun == nil {
		s.lastRun = &now
	}
	defer func() {
		s.lastRun = &now
	}()

	s.logger.Debug("Last run: ", s.lastRun.UTC())

	feedParser := gofeed.NewParser()

	for _, feed := range s.feeds {
		flogger := s.logger.With("id", feed.ID)

		parsed, err := feedParser.ParseURLWithContext(feed.URL, ctx)
		if err != nil {
			s.logger.Errorw("Failed to parse feed", "url", feed.URL, "error", err)
			continue
		}

		var feedTime *time.Time
		if parsed.PublishedParsed != nil {
			feedTime = parsed.PublishedParsed
		} else if parsed.UpdatedParsed != nil {
			feedTime = parsed.UpdatedParsed
		}

		flogger = flogger.With("time", feedTime)

		if feedTime != nil && feedTime.Before(*s.lastRun) {
			flogger.Debug("Feed has no new items")
			continue
		}

		flogger.Debugf("Processing...")

		for _, item := range parsed.Items {
			if item.PublishedParsed == nil {
				flogger.Warn("Item has no published time")
				continue
			}
			if item.PublishedParsed.Before(*s.lastRun) {
				continue
			}
			flogger.With(
				"guid", item.GUID,
				"published", item.Published,
				"authors", item.Authors,
			).Infof("%s ...", item.Title[:10])
		}
	}
}
