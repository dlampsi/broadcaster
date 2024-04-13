package memory

import (
	"broadcaster/storages"
	"broadcaster/structs"
	"context"
	"errors"
	"sync"

	"go.uber.org/zap"
)

type Storage struct {
	logger     *zap.SugaredLogger
	mu         *sync.RWMutex
	feeds      map[string]structs.RssFeed
	feedsItems map[string]structs.RssFeedItem
}

// Creates new in-memory storage.
func NewStorage(opts ...Option) *Storage {
	s := &Storage{
		logger:     zap.NewNop().Sugar(),
		mu:         &sync.RWMutex{},
		feeds:      make(map[string]structs.RssFeed),
		feedsItems: make(map[string]structs.RssFeedItem),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

type Option func(*Storage)

func WithLogger(logger *zap.SugaredLogger) Option {
	return func(s *Storage) {
		s.logger = logger
	}
}

// ------------------------------------------------------------------------------------------------

type Feeds struct {
	st *Storage
}

func (s *Storage) Feeds() storages.FeedsStorage {
	return &Feeds{st: s}
}

// Interface conformance assertion
var _ storages.FeedsStorage = &Feeds{}

func (s *Feeds) List(ctx context.Context) ([]structs.RssFeed, error) {
	var result []structs.RssFeed

	for _, feed := range s.st.feeds {
		result = append(result, feed)
	}
	return result, nil
}

func (s *Feeds) Find(ctx context.Context, req storages.FeedsStorageFindRequest) (*structs.RssFeed, error) {
	if feed, exists := s.st.feeds[req.Id]; exists {
		return &feed, nil
	}
	return nil, errors.New("Feed not found")
}

func (s *Feeds) Delete(ctx context.Context, req storages.FeedsStorageDeleteRequest) error {
	s.st.mu.Lock()
	defer s.st.mu.Unlock()

	if _, exists := s.st.feeds[req.Id]; !exists {
		return errors.New("Feed not found")
	}
	delete(s.st.feeds, req.Id)

	return nil
}

func (s *Feeds) Update(ctx context.Context, req storages.FeedsStorageUpdateRequest) (*structs.RssFeed, error) {
	return nil, nil
}

// ------------------------------------------------------------------------------------------------

type FeedItems struct {
	st *Storage
}

func (s *Storage) FeedItems() storages.FeedItemsStorage {
	return &FeedItems{st: s}
}

// Interface conformance assertion
var _ storages.FeedItemsStorage = &FeedItems{}

func (s *FeedItems) Find(ctx context.Context, req storages.FeedItemsStorageFindRequest) (*structs.RssFeedItem, error) {
	if feedItem, exists := s.st.feedsItems[req.Id]; exists {
		return &feedItem, nil
	}
	return nil, storages.ItemNotFoundError
}

func (s *FeedItems) Create(ctx context.Context, req storages.FeedItemsCreateRequest) (*structs.RssFeedItem, error) {
	s.st.mu.Lock()
	s.st.feedsItems[req.Id] = req.ToRssFeedItem()
	s.st.mu.Unlock()

	return s.Find(ctx, storages.FeedItemsStorageFindRequest{Id: req.Id})
}

func (s *FeedItems) List(ctx context.Context, req storages.FeedItemsListRequest) ([]structs.RssFeedItem, error) {
	var result []structs.RssFeedItem
	for _, feedItem := range s.st.feedsItems {
		result = append(result, feedItem)
	}
	return result, nil
}

func (s *FeedItems) Update(ctx context.Context, req storages.FeedItemsUpdateRequest) (*structs.RssFeedItem, error) {
	return nil, storages.NotImplementedError
}

func (s *FeedItems) Delete(ctx context.Context, req storages.FeedItemsStorageDeleteRequest) error {
	if _, exists := s.st.feedsItems[req.Id]; !exists {
		return storages.ItemNotFoundError
	}
	s.st.mu.Lock()
	delete(s.st.feedsItems, req.Id)
	s.st.mu.Unlock()
	return nil
}
