package translator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_GoogleApiTranslator(t *testing.T) {
	tr := NewGoogleApiTranslator()
	require.NotNil(t, tr)

	t.Run("Empty text", func(t *testing.T) {
		resp, err := tr.Translate(context.TODO(), TranlsationRequest{
			From: "en",
			To:   "fr",
			Text: []string{},
		})
		require.Error(t, err, "Empty text should return error")
		require.Nil(t, resp, "Repsonse should be nil on error")
	})
	t.Run("NotExistsLang", func(t *testing.T) {
		resp, err := tr.Translate(context.TODO(), TranlsationRequest{
			From: "en",
			To:   "fakelang",
			Text: []string{"Car!"},
		})
		require.Error(t, err)
		require.Nil(t, resp, "Repsonse should be nil on error")
	})
	t.Run("Valid", func(t *testing.T) {
		resp, err := tr.Translate(context.TODO(), TranlsationRequest{
			From: "en",
			To:   "fr",
			Text: []string{"Car!", "Today is Monday"},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, "Voiture!", resp.Title)
		require.Equal(t, "C'est lundi aujourd'hui", resp.Description)
	})
	t.Run("CustomFmt", func(t *testing.T) {
		resp, err := tr.Translate(context.TODO(), TranlsationRequest{
			From: "en",
			To:   "fr",
			Text: []string{"Car | Dog", "Monday"},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, "Voiture | Chien", resp.Title)
		require.Equal(t, "Lundi", resp.Description)
	})
}
