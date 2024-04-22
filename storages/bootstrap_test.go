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

		require.Len(t, feed.Translations, 1)
		ts := feed.Translations[0]
		require.Equal(t, "en", ts.To)
	})
}
