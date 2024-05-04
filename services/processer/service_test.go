package processer

import (
	"broadcaster/services/processer/translator"
	"broadcaster/storages/memory"
	"broadcaster/structs"
	"broadcaster/utils/logging"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Service for tests. Contains mock translator and disabled notifications.
var tservice *Service

func TestMain(m *testing.M) {
	os.Exit(testMainWrapper(m))
}

func testMainWrapper(m *testing.M) int {
	tlogger := logging.NewLogger("fatal", logging.FormatPretty)

	tstorage := memory.NewStorage(memory.WithLogger(tlogger))

	os.Setenv("BCTR_TRANSLATOR_TYPE", "mock")
	os.Setenv("BCTR_MUTE_NOTIFICATIONS", "true")

	s, err := NewService(tstorage, WithLogger(tlogger))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	tservice = s

	return m.Run()
}

func Test_Service_parseRssFeed(t *testing.T) {
	ctx := context.TODO()

	t.Run("nonExistsRss", func(t *testing.T) {
		feed := structs.RssFeed{
			Id:  "nonExistsRss",
			URL: "https://fakeurl.local/rss.xml",
		}
		items, err := tservice.parseRssFeed(ctx, feed, 2*time.Second)
		require.Error(t, err)
		require.Nil(t, items)
	})
	t.Run("validRss", func(t *testing.T) {
		feed := structs.RssFeed{
			Id:       "validRss",
			Source:   "ForTesting",
			Category: "Dummy",
			URL:      "https://www.hs.fi/rss/kaupunki.xml",
			Language: "fi",
		}
		items, err := tservice.parseRssFeed(ctx, feed, 10*time.Second)
		require.NoError(t, err)
		require.NotNil(t, items)

		ti := items[0]
		require.NotEmpty(t, ti.Id, "Item ID should be fetched from the source RSS feed item")
		require.Equal(t, feed.Id, ti.FeedId)
		require.Equal(t, feed.Source, ti.Source)
		require.NotEmpty(t, ti.Link)
		require.Equal(t, feed.Language, ti.Language)
		require.NotEmpty(t, ti.PubDate)
		require.Empty(t, ti.Processed, "Processed should be empty on a parsing step")
	})
	t.Run("withItemsLimits", func(t *testing.T) {
		feed := structs.RssFeed{
			Id:         "withItemsLimits",
			Source:     "ForTesting",
			Category:   "Dummy",
			URL:        "https://www.hs.fi/rss/kaupunki.xml",
			ItemsLimit: 1,
		}
		items, err := tservice.parseRssFeed(ctx, feed, 10*time.Second)
		require.NoError(t, err)
		require.Equal(t, 1, len(items), "Items limit should be respected")
		require.NotEmpty(t, items[0])
	})
}

func Test_Service_translateItems(t *testing.T) {
	ctx := context.TODO()

	feed := structs.RssFeed{
		Id:       "validRss",
		Source:   "ForTesting",
		Category: "Dummy",
		URL:      "https://fakeurl.local/rss.xml",
		Language: "en",
		Notifications: []structs.RssFeedNotification{
			{
				Type:      "slack",
				To:        []string{"slack1", "slack2"},
				Translate: structs.RssFeedTranslation{To: "fr"},
			},
			{
				Type:      "slack",
				To:        []string{"slack3"},
				Translate: structs.RssFeedTranslation{To: "fr"},
			},
		},
	}

	items := []structs.RssFeedItem{
		{
			Id:     "translationError",
			FeedId: "validRss",
			Link:   translator.MockURLForError,
		},
		{
			Id:       "sameLanguage",
			FeedId:   "validRss",
			Language: "fr",
		},
		{
			Id:          "validTranslation",
			FeedId:      "validRss",
			Source:      "ForTesting",
			Title:       "Hello World",
			Description: "This is a test",
			Link:        "https://www.example.com",
		},
	}

	tservice.translateItems(ctx, feed, items...)

	// TODO: Create separate test for this
	// require.Equal(
	// 	t, len(feed.GetTranslatonsLang()), len(tservice.translations),
	// 	"Only requested languages translates items should be stored in the cache",
	// )

	require.Nil(t, tservice.getTranslation("translationError", "en"), "Item should not be stored in the cache on error")
	require.Nil(t, tservice.getTranslation("sameLanguage", "en"), "Item should not be stored in the cache on error")
	require.NotNil(t, tservice.getTranslation("validTranslation", "en"), "Item should be stored in the cache on error")
}

func Test_Service_translateItem(t *testing.T) {
	ctx := context.TODO()

	t.Run("translationError", func(t *testing.T) {
		item := &structs.RssFeedItem{
			Id:   "translationError",
			Link: translator.MockURLForError,
		}
		err := tservice.translateItem(ctx, item, "fi", "en")
		require.Error(t, err)
	})
	t.Run("validTranslation", func(t *testing.T) {
		item := &structs.RssFeedItem{
			Id:          "validTranslation",
			FeedId:      "validRss",
			Source:      "ForTesting",
			Title:       "Hello World",
			Description: "This is a test",
			Link:        "https://www.example.com",
			Language:    "fi",
			PubDate:     time.Now(),
		}
		err := tservice.translateItem(ctx, item, "fi", "en")
		require.NoError(t, err)
	})
}
