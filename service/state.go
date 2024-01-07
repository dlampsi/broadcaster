package service

import (
	"broadcaster/structs"
	"context"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
	"go.uber.org/zap"
)

type state struct {
	logger *zap.SugaredLogger
	mu     *sync.RWMutex
	items  map[string]stateItem
}

type stateItem struct {
	Saved     time.Time
	Published *time.Time
}

func newState(logger *zap.SugaredLogger) *state {
	s := &state{
		logger: logger.Named("state"),
		mu:     &sync.RWMutex{},
		items:  make(map[string]stateItem),
	}
	return s
}

func getId(feed structs.FeedConfig, item *gofeed.Item) string {
	return feed.GetId() + "." + item.GUID
}

func (s *state) set(feed structs.FeedConfig, item *gofeed.Item) {
	now := time.Now().UTC()
	s.mu.Lock()
	s.items[getId(feed, item)] = stateItem{
		Published: item.PublishedParsed,
		Saved:     now,
	}
	s.mu.Unlock()
}

func (s *state) getPubTime(feed structs.FeedConfig, item *gofeed.Item) *time.Time {
	data, exists := s.items[getId(feed, item)]
	if !exists {
		return nil
	}
	return data.Published
}

func (s *state) cleanup(ctx context.Context, ttl time.Duration) (int, error) {
	deadline := time.Now().UTC().Add(-ttl)

	s.logger.Debugf("Deadline: %v (-%v)", deadline, ttl)

	var deleted int

	s.mu.Lock()
	defer s.mu.Unlock()

	for k, v := range s.items {
		ilogger := s.logger.With("id", k, "saved", v.Saved)

		if !v.Saved.Before(deadline) {
			ilogger.Debug("Skipping item")
			continue
		}

		ilogger.Debug("Removing item")
		delete(s.items, k)
		deleted++
	}

	s.logger.Debug("State size: ", len(s.items))

	return deleted, nil
}
