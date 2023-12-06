package config

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigFileLoad(t *testing.T) {
	ctx := context.Background()

	// Current path
	cpath, err := os.Getwd()
	require.NoError(t, err)
	require.NotEmpty(t, cpath)

	t.Run("BadURL", func(t *testing.T) {
		_, err := Load(ctx, "./testdata/config.yml")
		require.Error(t, err)
		require.Equal(t, UnknownSchemeError, err)
	})

	t.Run("BadFormat", func(t *testing.T) {
		_, err := Load(ctx, fmt.Sprintf("file:///%s/testdata/config_bad.yml", cpath))
		require.Error(t, err)
		require.NotEqual(t, UnknownSchemeError, err)
	})

	t.Run("Ok", func(t *testing.T) {

		cfg, err := Load(ctx, fmt.Sprintf("file:///%s/testdata/config.yml", cpath))
		require.NoError(t, err)
		require.NotNil(t, cfg)

		require.Len(t, cfg.Feeds, 1)
		feed := cfg.Feeds[0]
		require.NotEmpty(t, feed.Source)
		require.NotEmpty(t, feed.Category)
		require.NotEmpty(t, feed.URL)
		require.NotEmpty(t, feed.Translates)
	})
}
