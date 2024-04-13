package storages

import (
	"broadcaster/structs"
	"context"
	"errors"
	"time"
)

var (
	NotImplementedError error = errors.New("Not implemented")
	ItemNotFoundError   error = errors.New("Item not found")
)

type FeedsStorage interface {
	List(ctx context.Context) ([]structs.RssFeed, error)
	Find(ctx context.Context, req FeedsStorageFindRequest) (*structs.RssFeed, error)
	Delete(ctx context.Context, req FeedsStorageDeleteRequest) error
	Update(ctx context.Context, req FeedsStorageUpdateRequest) (*structs.RssFeed, error)
}

type FeedsStorageFindRequest struct {
	Id string
}

type FeedsStorageDeleteRequest struct {
	Id string
}

type FeedsStorageUpdateRequest struct {
	Id         string
	URL        string
	Language   string
	ItemsLimit int
	Notify     []structs.RssFeedNotification
	Translates []structs.RssFeedTranslation
}

type FeedItemsStorage interface {
	Create(ctx context.Context, req FeedItemsCreateRequest) (*structs.RssFeedItem, error)
	List(ctx context.Context, req FeedItemsListRequest) ([]structs.RssFeedItem, error)
	Find(ctx context.Context, req FeedItemsStorageFindRequest) (*structs.RssFeedItem, error)
	Delete(ctx context.Context, req FeedItemsStorageDeleteRequest) error
	Update(ctx context.Context, req FeedItemsUpdateRequest) (*structs.RssFeedItem, error)
}

type FeedItemsCreateRequest struct {
	Id          string
	FeedId      string
	Source      string
	Categories  []string
	Title       string
	Description string
	PubDate     time.Time
	Processed   time.Time
	Link        string
	Language    string
}

func (r FeedItemsCreateRequest) ToRssFeedItem() structs.RssFeedItem {
	return structs.RssFeedItem{
		Id:          r.Id,
		FeedId:      r.FeedId,
		Source:      r.Source,
		Categories:  r.Categories,
		Title:       r.Title,
		Description: r.Description,
		PubDate:     r.PubDate,
		Processed:   r.Processed,
		Link:        r.Link,
		Language:    r.Language,
	}
}

type FeedItemsStorageFindRequest struct {
	Id string
}

type FeedItemsStorageDeleteRequest struct {
	Id string
}

type FeedItemsListRequest struct {
	Limit      int
	Sources    []string
	Categories []string
	Languages  []string
	PubDate    *time.Time
}

type FeedItemsUpdateRequest struct {
	Id          string
	Categories  []string
	Title       string
	Description string
	PubDate     time.Time
	Processed   time.Time
	Link        string
	Language    string
}
