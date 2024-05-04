package structs

import (
	"slices"
	"time"
)

type RssFeed struct {
	Id            string
	Source        string
	Category      string
	URL           string
	Language      string
	ItemsLimit    int
	Notifications []RssFeedNotification
}

type RssFeedNotification struct {
	Type      string
	To        []string
	Muted     bool
	Translate RssFeedTranslation
}

type RssFeedTranslation struct {
	From string
	To   string
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

// Returns a list of languages to which the feed items should be translated.
func (c RssFeed) GetTranslatonsLang() []string {
	var result []string
	for _, n := range c.Notifications {
		if n.Translate.To != "" && !slices.Contains(result, n.Translate.To) {
			result = append(result, n.Translate.To)
		}
	}
	return result
}
