package service

import (
	"broadcaster/structs"
	"broadcaster/utils/info"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/mmcdole/gofeed"
	"go.uber.org/zap"
)

type Service struct {
	cfg        *Config
	translator Translator
	feeds      []structs.FeedConfig
	logger     *zap.SugaredLogger
	notifiers  map[string]Notifier
	lastRun    *time.Time // Timestamp of the last run in UTC
	state      *state
}

type Option func(*Service)

func WithLogger(l *zap.SugaredLogger) Option {
	return func(s *Service) { s.logger = l }
}

func WithFeeds(feeds ...structs.FeedConfig) Option {
	return func(s *Service) {
		s.feeds = append(s.feeds, feeds...)
	}
}

func WithConfig(c *Config) Option {
	return func(s *Service) {
		if c.TranslatorType != "" {
			s.cfg.TranslatorType = c.TranslatorType
		}
		if c.GoogleCloudProjectId != "" {
			s.cfg.GoogleCloudProjectId = c.GoogleCloudProjectId
		}
		if c.TelegramBotToken != "" {
			s.cfg.TelegramBotToken = c.TelegramBotToken
		}
		if c.BackfillHours > 0 {
			s.cfg.BackfillHours = c.BackfillHours
		}
		if c.MuteNotifications {
			s.cfg.MuteNotifications = c.MuteNotifications
		}
	}
}

func New(opts ...Option) (*Service, error) {
	var cfg Config
	if err := envconfig.Process(info.EnvPrefix, &cfg); err != nil {
		return nil, fmt.Errorf("Failed to load service config: %w", err)
	}

	s := &Service{
		cfg:       &cfg,
		logger:    zap.NewNop().Sugar(),
		notifiers: map[string]Notifier{},
	}

	for _, opt := range opts {
		opt(s)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("Invalid configuration: %w", err)
	}

	switch s.cfg.TranslatorType {
	case "mock", "raw":
		s.translator = NewMockTranslator()
	case "google_cloud":
		cfg := &GoogleCloudTranslatorConfig{
			ProjectId: s.cfg.GoogleCloudProjectId,
		}
		if s.cfg.GoogleCloudCreds != "" {
			s.logger.Debug("Using Google Cloud credentials from env")
			cfg.CredsJson = []byte(s.cfg.GoogleCloudCreds)
		}
		s.translator = NewGoogleCloudTranslator(cfg)
	default:
		return nil, fmt.Errorf("Unsupported translator type '%s'", s.cfg.TranslatorType)
	}

	if cfg.TelegramBotToken != "" {
		s.logger.Debug("Loading Telegram notifier")
		tn, err := NewTelegramNotifier(cfg.TelegramBotToken, s.logger)
		if err != nil {
			return nil, fmt.Errorf("Failed to init a telegram notifier: %w", err)
		}
		s.notifiers["telegram"] = tn
	}

	if cfg.SlackApiToken != "" {
		s.logger.Debug("Loading Slack notifier")
		s.notifiers["slack"] = NewSlackNotifier(cfg.SlackApiToken, s.logger)
	}

	s.state = newState(s.logger)

	return s, nil
}

func (s *Service) Process(ctx context.Context) error {
	s.logger.Debug("Starting feeds processing")
	defer s.logger.Debug("Feeds processing is done")

	now := time.Now().UTC()

	if s.cfg.BackfillHours > 0 {
		now = now.Add(-time.Duration(s.cfg.BackfillHours) * time.Hour)
	}

	if s.lastRun == nil {
		s.lastRun = &now
	}
	s.logger.Debug("Last run: ", s.lastRun)

	defer func() {
		s.lastRun = &now
	}()

	if s.cfg.MuteNotifications {
		s.logger.Warn("Notifications are muted")
	}

	for _, f := range s.feeds {
		if err := s.processFeed(ctx, f); err != nil {
			s.logger.Errorw("Failed to process feed", "err", err.Error())
		}
	}

	return nil
}

func (s *Service) processFeed(ctx context.Context, feed structs.FeedConfig) error {
	flogger := s.logger.With("src", feed.GetId())

	fp := gofeed.NewParser()
	parsed, err := fp.ParseURLWithContext(feed.URL, ctx)
	if err != nil {
		return fmt.Errorf("Failed to parse feed: %w", err)
	}
	flogger.Debugf("Feed '%s' successfully parsed", parsed.Title)

	limit := feed.ItemsLimit
	if limit == 0 {
		limit = 5
	}

	var items []*structs.FeedItem

	for _, item := range parsed.Items[:limit] {
		ilogger := s.logger.With("id", item.GUID, "link", item.Link, "src", feed.GetId())

		if pt := s.state.getPubTime(feed, item); pt != nil {
			ilogger.Debug("Feed item already in state, skipping")
			if !pt.Equal(*item.PublishedParsed) {
				ilogger.Debug("Item has been updated since publication")
				continue
			}
			continue
		}

		if s.lastRun != nil && item.PublishedParsed.Before(*s.lastRun) {
			ilogger.With("published", item.Published).Debug("Skipping old item")
			continue
		}

		/*
			Adding a raw feed data if no translations are required
			otherwise add translated data for each language
		*/
		if len(feed.Translates) == 0 {
			items = append(items, &structs.FeedItem{
				Title:       item.Title,
				Description: item.Description,
				Link:        item.Link,
				Source:      feed.Source,
			})
			s.state.set(feed, item)
			ilogger.Info("Item has no translations, added as is")
			continue
		}

		for _, translate := range feed.Translates {
			tlogger := ilogger.With("translate", feed.Language+"."+translate.To)

			resp, err := s.translator.Translate(ctx, TranlsationRequest{
				Link: item.Link,
				From: feed.Language,
				To:   translate.To,
				Text: []string{item.Title, item.Description},
			})
			if err != nil {
				tlogger.Errorw("Failed to translate item text", "err", err.Error())
				continue
			}
			resp.Source = feed.Source

			tlogger.Info("Item has been translated")

			items = append(items, resp)
		}

		s.state.set(feed, item)
	}

	var wg sync.WaitGroup

	for _, nc := range feed.Notify {
		if _, ok := s.notifiers[nc.Type]; !ok {
			flogger.Warnf(
				"Notifier '%s' isn't configured. You may not have specified a notification token env.",
				nc.Type,
			)
			continue
		}

		wg.Add(1)

		go func(c structs.FeedNotifyConfig) {
			defer wg.Done()

			for _, fi := range items {
				if err := s.notify(ctx, c, fi); err != nil {
					flogger.With("err", err.Error()).Errorf("Failed to notify with '%s'", c.Type)
				}
			}
		}(nc)
	}

	wg.Wait()

	return nil
}

func (s *Service) notify(ctx context.Context, cfg structs.FeedNotifyConfig, item *structs.FeedItem) error {
	var (
		notifier Notifier
		request  NotificationRequest
	)

	switch cfg.Type {
	case "telegram":
		notifier = s.notifiers["telegram"]

		request = NotificationRequest{
			To: cfg.To,
			Message: fmt.Sprintf(
				"*%s* \n\n%s\n\n[%s](%s)",
				item.Title,
				item.Description,
				item.Source,
				item.Link,
			),
		}
	case "slack":
		notifier = s.notifiers["slack"]

		request = NotificationRequest{
			To:     cfg.To,
			Source: item.Source,
			Message: fmt.Sprintf(
				"<%s|%s>\n\n%s",
				item.Link,
				item.Title,
				item.Description,
			),
		}
	default:
		return fmt.Errorf("Unsupported notification type '%s'", cfg.Type)
	}

	return notifier.Notify(ctx, request)
}

func (s *Service) CleanupState(ctx context.Context) error {
	s.logger.Debug("Cleaning up state")
	deleted, err := s.state.cleanup(ctx, time.Duration(s.cfg.StateTTL)*time.Second)
	if err != nil {
		return err
	}
	if deleted > 0 {
		s.logger.Infof("State was cleared from '%d' items", deleted)
	}
	s.logger.Debug("State cleanup is done")
	return nil
}
