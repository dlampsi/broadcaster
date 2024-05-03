package processer

import (
	"broadcaster/services/processer/notifier"
	"broadcaster/services/processer/translator"
	"broadcaster/storages"
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

type Storage interface {
	Feeds() storages.FeedsStorage
	FeedItems() storages.FeedItemsStorage
}

type Service struct {
	cfg        *Config
	logger     *zap.SugaredLogger
	storage    Storage
	translator translator.Translator
	notifiers  map[string]notifier.Notifier

	lastRun *time.Time // Timestamp of the last run in UTC
}

type Option func(*Service)

func WithLogger(logger *zap.SugaredLogger) Option {
	return func(s *Service) {
		s.logger = logger
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

func NewService(storage Storage, opts ...Option) (*Service, error) {
	var cfg Config
	if err := envconfig.Process(info.EnvPrefix, &cfg); err != nil {
		return nil, fmt.Errorf("Failed to load service config: %w", err)
	}

	svc := &Service{
		cfg:       &cfg,
		logger:    zap.NewNop().Sugar(),
		storage:   storage,
		notifiers: make(map[string]notifier.Notifier),
	}

	for _, opt := range opts {
		opt(svc)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("Invalid configuration: %w", err)
	}

	switch svc.cfg.TranslatorType {
	case "mock", "raw":
		svc.translator = translator.NewMockTranslator()
	case "google_cloud":
		cfg := &translator.GoogleCloudTranslatorConfig{
			ProjectId: svc.cfg.GoogleCloudProjectId,
		}
		if svc.cfg.GoogleCloudCreds != "" {
			svc.logger.Debug("Using Google Cloud credentials from env")
			cfg.CredsJson = []byte(svc.cfg.GoogleCloudCreds)
		}
		svc.translator = translator.NewGoogleCloudTranslator(cfg)
	default:
		return nil, fmt.Errorf("Unsupported translator type '%s'", svc.cfg.TranslatorType)
	}

	if cfg.TelegramBotToken != "" {
		svc.logger.Debug("Loading Telegram notifier")
		tn, err := notifier.NewTelegramNotifier(
			cfg.TelegramBotToken,
			svc.logger.Named("notifier").Named("telegram"),
		)
		if err != nil {
			return nil, fmt.Errorf("Failed to init a telegram notifier: %w", err)
		}
		svc.notifiers["telegram"] = tn
	}

	if cfg.SlackApiToken != "" {
		svc.logger.Debug("Loading Slack notifier")
		svc.notifiers["slack"] = notifier.NewSlackNotifier(
			cfg.SlackApiToken,
			svc.logger.Named("notifier").Named("slack"),
		)
	}

	return svc, nil
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
	defer func() {
		s.lastRun = &now
	}()

	s.logger.Debug("Last run: ", s.lastRun)

	if s.cfg.MuteNotifications {
		s.logger.Warn("Notifications are muted")
	}

	feeds, err := s.storage.Feeds().List(ctx)
	if err != nil {
		return fmt.Errorf("Failed to load feeds: %w", err)
	}
	for _, feed := range feeds {
		if err := s.processFeed(ctx, feed); err != nil {
			s.logger.With("feed_id", feed.Id).Errorw("Failed to process feed", "err", err.Error())
		}
	}

	return nil
}

func (s *Service) processFeed(ctx context.Context, feed structs.RssFeed) error {
	logger := s.logger.With("feed_id", feed.Id)

	if len(feed.Notifications) == 0 {
		logger.Debug("Feed has no notifications configured, skipping")
		return nil
	}

	logger.Debug("Parsing feed")

	fp := gofeed.NewParser()
	parsed, err := fp.ParseURLWithContext(feed.URL, ctx)
	if err != nil {
		return fmt.Errorf("Failed to parse feed: %w", err)
	}

	if feed.ItemsLimit == 0 {
		feed.ItemsLimit = 10
	}
	if feed.ItemsLimit > len(parsed.Items) {
		feed.ItemsLimit = len(parsed.Items)
	}

	var pitems []*structs.RssFeedItem

	for _, pi := range parsed.Items[:feed.ItemsLimit] {
		if i, err := s.processFeedItem(ctx, feed, pi); err != nil {
			logger.With("guid", pi.GUID).Errorw("Failed to process feed item", "err", err.Error())
		} else if i != nil {
			pitems = append(pitems, i)
		}
	}

	var wg sync.WaitGroup

	for _, nc := range feed.Notifications {
		if nc.Muted {
			logger.Debugf("Notification is muted for: '%s'", nc.Type)
			continue
		}

		nf, nfExists := s.notifiers[nc.Type]
		if !nfExists {
			logger.Warnf(
				"Notifier '%s' isn't configured. You may not have specified a notification token env.",
				nc.Type,
			)
			continue
		}

		wg.Add(1)

		go func(fn structs.RssFeedNotification, nf notifier.Notifier) {
			defer wg.Done()
			for _, fi := range pitems {
				notifyRequest := nf.NewRequest(fn, fi)
				if err := nf.Notify(ctx, notifyRequest); err != nil {
					logger.With("err", err.Error()).Errorf("Failed to notify with '%s'", fn.Type)
				}
			}
		}(nc, nf)
	}

	wg.Wait()

	return nil
}

func (s *Service) processFeedItem(ctx context.Context, feed structs.RssFeed, item *gofeed.Item) (*structs.RssFeedItem, error) {
	logger := s.logger.With("feed_id", feed.Id, "guid", item.GUID, "link", item.Link)

	logger.Debug("Searching for item in storage")

	findReq := storages.FeedItemsStorageFindRequest{
		Id: item.GUID,
	}
	fi, err := s.storage.FeedItems().Find(ctx, findReq)
	if err != nil && err != storages.ItemNotFoundError {
		return nil, fmt.Errorf("Failed to find feed item in storage: %w", err)
	}
	if fi != nil {
		logger.Debug("Feed item already in state, skipping")
		return nil, nil
	}

	if s.lastRun != nil && item.PublishedParsed.Before(*s.lastRun) {
		logger.Debug("Skipping feed item published before the last run")
		return nil, nil
	}

	/*
		Adding a raw feed data if no translations are required
		otherwise add translated data for each language
	*/

	processed := &structs.RssFeedItem{
		Id:          item.GUID,
		Source:      feed.Source,
		Categories:  item.Categories,
		Title:       item.Title,
		Description: item.Description,
		Link:        item.Link,
		Language:    feed.Language,
		PubDate:     *item.PublishedParsed,
	}

	for _, translate := range feed.Translations {
		tlogger := logger.With("translate", feed.Language+"."+translate.To)

		tlogger.Debug("Translating feed item")

		translateReq := translator.TranlsationRequest{
			Link: item.Link,
			From: feed.Language,
			To:   translate.To,
			Text: []string{item.Title, item.Description},
		}
		tresp, err := s.translator.Translate(ctx, translateReq)
		if err != nil {
			tlogger.Errorw("Failed to translate item text", "err", err.Error())
			continue
		}

		processed.Title = tresp.Title
		processed.Description = tresp.Description
		processed.Link = tresp.Link
	}

	logger.Debug("Saving feed item to storage")

	createReq := storages.FeedItemsCreateRequest{
		Id:          processed.Id,
		FeedId:      feed.Id,
		Source:      processed.Source,
		Categories:  processed.Categories,
		Title:       processed.Title,
		Description: processed.Description,
		PubDate:     processed.PubDate,
		Processed:   time.Now().UTC(),
		Link:        processed.Link,
		Language:    processed.Language,
	}
	rec, err := s.storage.FeedItems().Create(ctx, createReq)
	if err != nil {
		return processed, fmt.Errorf("Failed to save feed item: %w", err)
	}

	logger.Info("Feed item has been processed")

	return rec, nil
}
