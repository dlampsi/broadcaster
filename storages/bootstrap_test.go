package storages

import (
	"broadcaster/utils/logging"
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_GetFeedsFromConfig(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewLogger("fatal", "pretty")

	// Current path
	cpath, err := os.Getwd()
	require.NoError(t, err)
	require.NotEmpty(t, cpath)

	t.Run("BadURL", func(t *testing.T) {
		cfg, err := GetFeedsFromConfig(ctx, "./testdata/config.yml", logger)
		require.Error(t, err)
		require.Nil(t, cfg)
	})

	t.Run("BadFormat", func(t *testing.T) {
		cfg, err := GetFeedsFromConfig(ctx, fmt.Sprintf("file:///%s/testdata/bootstrap_bad.yml", cpath), logger)
		require.Error(t, err)
		require.Nil(t, cfg)
	})

	t.Run("ValidConfig", func(t *testing.T) {
		cfg, err := GetFeedsFromConfig(ctx, fmt.Sprintf("file:///%s/testdata/bootstrap_ok.yml", cpath), logger)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		require.Len(t, cfg, 1)
		feed := cfg[0]

		require.Equal(t, "Helsingin Sanomat", feed.Source)
		require.Equal(t, "City", feed.Category)
		require.Equal(t, "https://www.hs.fi/rss/kaupunki.xml", feed.URL)
		require.Equal(t, "fi", feed.Language)

		require.Len(t, feed.Notifications, 1)
		notification := feed.Notifications[0]
		require.NotEmpty(t, "en", notification.Type)
		require.NotEmpty(t, "en", notification.To)
		require.False(t, notification.Muted)
		require.Equal(t, "en", notification.Translate.To)
	})
}

func Test_FeedConfig_ToRssFeed(t *testing.T) {
	cfg := FeedConfig{
		Source:     "Dummy Feed",
		Category:   "Test",
		URL:        "https://example.com/rss.xml",
		Language:   "en",
		ItemsLimit: 10,
		Notifications: []FeedNotificationsConfig{
			{
				Type:  "email",
				To:    []string{"fake@email"},
				Muted: false,
				Translate: FeedTranslationsConfig{
					To: "fi",
				},
			},
			{
				Type:  "email",
				To:    []string{"fake2@email"},
				Muted: false,
				Translate: FeedTranslationsConfig{
					From: "eng",
					To:   "fi",
				},
			},
		},
	}
	feed := cfg.ToRssFeed()

	require.Equal(t, "DummyFeed.Test", feed.Id)
	require.Equal(t, cfg.Source, feed.Source)
	require.Equal(t, cfg.Category, feed.Category)
	require.Equal(t, cfg.URL, feed.URL)
	require.Equal(t, cfg.Language, feed.Language)
	require.Equal(t, cfg.ItemsLimit, feed.ItemsLimit)

	require.Equal(t, len(cfg.Notifications), len(feed.Notifications))
	require.Equal(t, cfg.Notifications[0].Type, feed.Notifications[0].Type)
	require.Equal(t, cfg.Notifications[0].To, feed.Notifications[0].To)
	require.Equal(t, cfg.Notifications[0].Muted, feed.Notifications[0].Muted)
	require.Equal(t, cfg.Notifications[0].Translate.To, feed.Notifications[0].Translate.To)
	require.Equal(t, cfg.Language, feed.Notifications[0].Translate.From)
	require.Equal(t, cfg.Notifications[1].Translate.From, feed.Notifications[1].Translate.From)
}

func Test_coalesce(t *testing.T) {
	require.Equal(t, "", coalesce())
	require.Equal(t, "", coalesce("", ""))
	require.Equal(t, "two", coalesce("", "two", "three"))
}
