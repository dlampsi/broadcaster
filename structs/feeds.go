package structs

import "time"

type RssFeed struct {
	Id            string
	Source        string
	Category      string
	URL           string
	Language      string
	ItemsLimit    int
	Notifications []RssFeedNotification
	Translations  []RssFeedTranslation
}

type RssFeedNotification struct {
	Type  string
	To    []string
	Muted bool
}

type RssFeedTranslation struct {
	To string
}

type RssFeedItem struct {
	Id          string
	FeedId      string
	Source      string
	Categories  []string
	Title       string
	Description string
	Link        string
	Language    string
	PubDate     time.Time // Publication date (from the feed)
	Processed   time.Time // When the item was processed by the service
}
