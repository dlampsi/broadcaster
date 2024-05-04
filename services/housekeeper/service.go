package housekeeper

import (
	"broadcaster/storages"
	"context"
	"time"

	"go.uber.org/zap"
)

type Service struct {
	logger  *zap.SugaredLogger
	storage Storage
}

type Storage interface {
	FeedItems() storages.FeedItemsStorage
}

type Option func(*Service)

func WithLogger(logger *zap.SugaredLogger) Option {
	return func(s *Service) {
		s.logger = logger
	}
}

func NewService(storage Storage, opts ...Option) (*Service, error) {
	svc := &Service{
		logger:  zap.NewNop().Sugar(),
		storage: storage,
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc, nil
}

func (s *Service) CleanupFeedItems(ctx context.Context, ttl time.Duration) error {
	items, err := s.storage.FeedItems().List(ctx, storages.FeedItemsListRequest{})
	if err != nil {
		return err
	}
	s.logger.Debugf("Found %d feed items", len(items))

	deadline := time.Now().UTC().Add(-ttl)
	s.logger.Debugf("Deadline: %v (-%v)", deadline, ttl)

	var deleted int

	for _, item := range items {
		ilogger := s.logger.With("item_id", item.Id, "saved", item.Processed, "feed_id", item.FeedId)
		if !item.Processed.Before(deadline) {
			ilogger.Debug("Skipping item cleanup")
			continue
		}

		ilogger.Debug("Removing item")

		delReq := storages.FeedItemsStorageDeleteRequest{
			Id: item.Id,
		}
		if err := s.storage.FeedItems().Delete(ctx, delReq); err != nil {
			ilogger.Error("Failed to cleanup item: ", err)
			continue
		}
		deleted++
	}

	s.logger.Debug("Cleaned items: ", deleted)

	return nil
}
