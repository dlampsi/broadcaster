package service

import (
	"a0feed/structs"
	"a0feed/utils/info"
	"context"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kelseyhightower/envconfig"
	"github.com/mmcdole/gofeed"
	"go.uber.org/zap"
)

type Service struct {
	cfg        *Config
	translator Translator
	feeds      []structs.FeedConfig
	logger     *zap.SugaredLogger
	tgbot      *tgbotapi.BotAPI
	lastRun    *time.Time // Timestamp of the last run in UTC
	state      *State
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
		cfg:    &cfg,
		logger: zap.NewNop().Sugar(),
		state:  NewState(),
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
		s.translator = NewGoogleCloudTranslator(s.cfg.GoogleCloudProjectId)
	default:
		return nil, fmt.Errorf("Unsupported translator type '%s'", s.cfg.TranslatorType)
	}

	tgbot, err := tgbotapi.NewBotAPI(s.cfg.TelegramBotToken)
	if err != nil {
		return nil, fmt.Errorf("Failed to init a telegram client: %w", err)
	}
	s.tgbot = tgbot

	return s, nil
}

func (s *Service) Process(ctx context.Context) error {
	s.logger.Info("Processing feeds data")
	defer s.logger.Info("Processing feeds data is done")

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

	for _, f := range s.feeds {
		if err := s.processFeed(ctx, f); err != nil {
			s.logger.Errorw("Failed to process feed", "err", err.Error())
		}
	}

	return nil
}

func (s *Service) processFeed(ctx context.Context, feed structs.FeedConfig) error {
	flogger := s.logger.With(zap.String("source", feed.Source+" | "+feed.Category))

	fp := gofeed.NewParser()
	parsed, err := fp.ParseURLWithContext(feed.URL, ctx)
	if err != nil {
		return fmt.Errorf("Failed to parse feed: %w", err)
	}

	flogger.With("title", parsed.Title).Infof("Processing feed")

	limit := feed.ItemsLimit
	if limit == 0 {
		limit = 5
	}

	for _, fi := range parsed.Items[:limit] {
		ilogger := flogger.With(
			zap.String("id", fi.GUID),
			zap.String("url", fi.Link),
			zap.String("published", fi.Published),
		)

		if pt := s.state.GetPubTime(feed, fi); pt != nil {
			ilogger.Debug("Feed item already in state, skipping")
			if !pt.Equal(*fi.PublishedParsed) {
				ilogger.Warnf("State item timestamp '%s' mismatch, source '%s'", pt, fi.PublishedParsed)
				continue
			}
			continue
		}

		if s.lastRun != nil && fi.PublishedParsed.Before(*s.lastRun) {
			ilogger.Debug("Skipping old feed item")
			continue
		}

		ilogger.Debug("Translating feed item")

		for _, ft := range feed.Translates {
			tlogger := ilogger.With("transate", fmt.Sprintf("%s > %s", ft.From, ft.To))

			resp, err := s.translator.Translate(ctx, TranlsationRequest{
				Link: fi.Link,
				From: ft.From,
				To:   ft.To,
				Text: []string{fi.Title, fi.Description},
			})
			if err != nil {
				tlogger.Errorw("Failed to translate item text", "err", err.Error())
				continue
			}
			resp.Source = feed.Source

			tlogger.Info("Feed item translated")

			if s.cfg.MuteNotifications {
				tlogger.Info("Notifications are muted")
				continue
			}

			for _, fn := range ft.Notify {
				if err := s.notify(ctx, fn, resp); err != nil {
					tlogger.With("err", err.Error()).Errorf("Failed to notify with '%s'", fn.Type)
				}
			}
		}

		s.state.Set(feed, fi)
	}

	return nil
}

func (s *Service) notify(ctx context.Context, cfg structs.FeedTranslateNotifyConfig, item *structs.TranslatedFeedItem) error {
	switch cfg.Type {
	case "telegram":
		msg := tgbotapi.NewMessage(
			cfg.ChatId,
			fmt.Sprintf(
				"*%s* \n\n%s\n\n[%s](%s)",
				item.Title,
				item.Description,
				item.Source,
				item.Link,
			),
		)
		msg.ParseMode = "markdown"
		msg.DisableWebPagePreview = false

		if _, err := s.tgbot.Send(msg); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unsupported notification type '%s'", cfg.Type)
	}
	return nil
}
