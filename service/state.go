package service

import (
	"a0feed/structs"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
)

type State struct {
	mu    *sync.RWMutex
	items map[string]*time.Time
}

func NewState() *State {
	s := &State{
		mu:    &sync.RWMutex{},
		items: make(map[string]*time.Time),
	}
	return s
}

func getId(feed structs.FeedConfig, item *gofeed.Item) string {
	return feed.Source + "." + feed.Category + "." + item.GUID
}

func (s *State) Set(feed structs.FeedConfig, item *gofeed.Item) {
	s.mu.Lock()
	s.items[getId(feed, item)] = item.PublishedParsed
	s.mu.Unlock()
}

func (s *State) GetPubTime(feed structs.FeedConfig, item *gofeed.Item) *time.Time {
	data, exists := s.items[getId(feed, item)]
	if !exists {
		return nil
	}
	return data
}
