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
	mu         *sync.RWMutex
	// In-memory cache for translated items. item_uid -> language -> item
	translations map[string]map[string]structs.RssFeedItem
	// Timestamp of the last run in UTC
	lastRun *time.Time
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
		cfg:          &cfg,
		logger:       zap.NewNop().Sugar(),
		storage:      storage,
		notifiers:    make(map[string]notifier.Notifier),
		mu:           &sync.RWMutex{},
		translations: make(map[string]map[string]structs.RssFeedItem),
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
	case "google_api":
		svc.translator = translator.NewGoogleApiTranslator()
	default:
		return nil, fmt.Errorf("Unsupported translator type '%s'", svc.cfg.TranslatorType)
	}

	if cfg.TelegramBotToken != "" && !cfg.MuteNotifications {
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

/*
For each feed
*/
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

	s.logger.Debug("Clearing translations cache; Current size: ", len(s.translations))
	s.mu.Lock()
	s.translations = make(map[string]map[string]structs.RssFeedItem)
	s.mu.Unlock()

	feeds, err := s.storage.Feeds().List(ctx)
	if err != nil {
		return fmt.Errorf("Failed to load feeds: %w", err)
	}
	s.logger.Debugf("Loaded '%d' feeds from storage", len(feeds))

	var wg sync.WaitGroup
	wg.Add(len(feeds))

	for _, f := range feeds {
		go func(feed structs.RssFeed) {
			defer wg.Done()
			if err := s.processFeed(ctx, feed); err != nil {
				s.logger.With("feed_id", feed.Id).Errorw("Failed to process feed", "err", err.Error())
			}
		}(f)
	}

	wg.Wait()

	return nil
}

func (s *Service) processFeed(ctx context.Context, feed structs.RssFeed) error {
	logger := s.logger.With("feed_id", feed.Id)

	if len(feed.Notifications) == 0 {
		logger.Debug("Feed has no notifications configured, skipping")
		return nil
	}

	items, err := s.parseRssFeed(ctx, feed, 120*time.Second)
	if err != nil {
		return fmt.Errorf("Failed to parse feed: %w", err)
	}
	logger.Debug("Parsed feed items: ", len(items))

	items = s.filterItems(ctx, feed, items...)
	logger.Debug("Feed items after filtering: ", len(items))

	s.translateItems(ctx, feed, items...)

	var wg sync.WaitGroup
	wg.Add(len(feed.Notifications))
	for _, nn := range feed.Notifications {
		go s.notifyFeed(ctx, &wg, feed, nn, items...)
	}
	wg.Wait()

	s.storeItems(ctx, feed, items...)

	return nil
}

// Parses the RSS feed and returns a list of converted items.
func (s *Service) parseRssFeed(ctx context.Context, feed structs.RssFeed, timeout time.Duration) ([]structs.RssFeedItem, error) {
	logger := s.logger.With("feed_id", feed.Id)

	var items []structs.RssFeedItem

	logger.Debug("Parsing feed")

	feedParser := gofeed.NewParser()

	pCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	parsedFeed, err := feedParser.ParseURLWithContext(feed.URL, pCtx)
	if err != nil {
		return nil, err
	}

	limit := feed.ItemsLimit
	if limit == 0 {
		limit = 10
	}
	if limit > len(parsedFeed.Items) {
		limit = len(parsedFeed.Items)
	}

	logger.Debugf("Parsed %d items; Limit: %d", len(parsedFeed.Items), limit)

	for _, item := range parsedFeed.Items[:limit] {
		items = append(items, structs.RssFeedItem{
			Id:          item.GUID,
			FeedId:      feed.Id,
			Source:      feed.Source,
			Categories:  item.Categories,
			Title:       item.Title,
			Description: item.Description,
			Link:        item.Link,
			Language:    feed.Language,
			PubDate:     *item.PublishedParsed,
		})
	}

	return items, nil
}

// Translates feed items and stores the translations in the cache.
func (s *Service) translateItems(ctx context.Context, feed structs.RssFeed, items ...structs.RssFeedItem) {
	logger := s.logger.With("feed_id", feed.Id)

	for _, lang := range feed.GetTranslatonsLang() {
		for _, item := range items {
			ilogger := logger.With("item_id", item.Id)

			if item.Language == lang {
				ilogger.Debug("Item is already in desired language")
				continue
			}

			if ti := s.getTranslation(item.Id, lang); ti != nil {
				ilogger.Debug("Item translation found in cache")
				continue
			}

			if err := s.translateItem(ctx, &item, feed.Language, lang); err != nil {
				ilogger.Errorw("Failed to translate item", "err", err.Error())
				continue
			}

			s.saveTranslation(item, lang)
		}
	}
}

func (s *Service) translateItem(ctx context.Context, item *structs.RssFeedItem, from, to string) error {
	logger := s.logger.With("item_id", item.Id, "translate", from+"->"+to)

	logger.Debug("Translating item")

	req := translator.TranlsationRequest{
		Link: item.Link,
		From: from,
		To:   to,
		Text: []string{item.Title, item.Description},
	}
	resp, err := s.translator.Translate(ctx, req)
	if err != nil {
		return err
	}

	item.Title = resp.Title
	item.Description = resp.Description
	item.Link = resp.Link

	return nil
}

// Checks if the items are not processed yet and whether they pub data is newer than the last run.
func (s *Service) filterItems(ctx context.Context, feed structs.RssFeed, items ...structs.RssFeedItem) []structs.RssFeedItem {
	logger := s.logger.With("feed_id", feed.Id)

	var filtered []structs.RssFeedItem

	for _, item := range items {
		ilogger := logger.With("item_id", item.Id)

		findReq := storages.FeedItemsStorageFindRequest{
			Id: item.Id,
		}
		fi, err := s.storage.FeedItems().Find(ctx, findReq)
		if err != nil && err != storages.ItemNotFoundError {
			ilogger.With("err", err.Error()).Error("Failed to find feed item in storage")
		}
		if fi != nil {
			ilogger.Debug("Item already in state, skipping")
			continue
		}

		if s.lastRun != nil && item.PubDate.Before(*s.lastRun) {
			ilogger.Debug("Skipping item published before the last run")
			continue
		}

		filtered = append(filtered, item)
	}

	return filtered
}

func (s *Service) notifyFeed(ctx context.Context, wg *sync.WaitGroup, feed structs.RssFeed, nfn structs.RssFeedNotification, items ...structs.RssFeedItem) {
	defer wg.Done()

	logger := s.logger.With("feed_id", feed.Id, "notify_type", nfn.Type)

	if nfn.Muted {
		logger.Debugf("Notification is muted for: '%s'", nfn.Type)
		return
	}

	nfr, exists := s.notifiers[nfn.Type]
	if !exists {
		logger.Warnf(
			"Notifier '%s' isn't configured. You may not have specified a notification token env.",
			nfn.Type,
		)
		return
	}

	for _, item := range items {
		ilogger := logger.With("item_id", item.Id)

		if nfn.Translate.To != "" && nfn.Translate.To != item.Language {
			tItem := s.getTranslation(item.Id, nfn.Translate.To)
			if tItem != nil {
				ilogger.Debug("Item translation found in cache")
				item = *tItem
			} else {
				logger.Warn("Item translation not found in cache. Translating on the fly...")

				if err := s.translateItem(ctx, &item, item.Language, nfn.Translate.To); err != nil {
					ilogger.With("err", err.Error()).Errorf("Failed to translate item on the fly")
				}
			}
		}

		ilogger.Info("Sending notification")

		req := nfr.NewRequest(nfn, &item)
		if err := nfr.Notify(ctx, req); err != nil {
			logger.With("item_id", item.Id, "err", err.Error()).
				Errorf("Failed to notify with '%s'", nfn.Type)
		}
	}
}

func (s *Service) storeItems(ctx context.Context, feed structs.RssFeed, items ...structs.RssFeedItem) {
	logger := s.logger.With("feed_id", feed.Id)
	for _, item := range items {
		ilogger := logger.With("item_id", item.Id)

		ilogger.Debug("Storing item in storage")

		req := storages.FeedItemsCreateRequest{
			Id:          item.Id,
			FeedId:      feed.Id,
			Source:      item.Source,
			Categories:  item.Categories,
			Title:       item.Title,
			Description: item.Description,
			PubDate:     item.PubDate,
			Processed:   time.Now().UTC(),
			Link:        item.Link,
			Language:    item.Language,
		}
		if _, err := s.storage.FeedItems().Create(ctx, req); err != nil {
			ilogger.With("err", err.Error()).Error("Failed to save item to storage")
		}
	}
}

func (s *Service) saveTranslation(item structs.RssFeedItem, lang string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.translations[item.Id]; !exists {
		s.translations[item.Id] = make(map[string]structs.RssFeedItem)
	}
	s.translations[item.Id][lang] = item
}

func (s *Service) getTranslation(itemId string, lang string) *structs.RssFeedItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.translations[itemId]; ok {
		if item, ok := s.translations[itemId][lang]; ok {
			return &item
		}
	}
	return nil
}
