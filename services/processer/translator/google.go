package translator

import (
	"context"
	"fmt"

	googleTranslate "cloud.google.com/go/translate/apiv3"
	googleTranslatePb "cloud.google.com/go/translate/apiv3/translatepb"
	googleOption "google.golang.org/api/option"
)

// Validates interface compliance
var _ Translator = (*GoogleCloudTranslator)(nil)

type GoogleCloudTranslatorConfig struct {
	ProjectId string
	CredsJson []byte // Optional credentials in JSON format
}

type GoogleCloudTranslator struct {
	cfg *GoogleCloudTranslatorConfig
}

func NewGoogleCloudTranslator(cfg *GoogleCloudTranslatorConfig) *GoogleCloudTranslator {
	return &GoogleCloudTranslator{
		cfg: cfg,
	}
}

func (t *GoogleCloudTranslator) Translate(ctx context.Context, r TranlsationRequest) (*TranlsationResponce, error) {
	var clOpts []googleOption.ClientOption

	if len(t.cfg.CredsJson) > 0 {
		clOpts = append(clOpts, googleOption.WithCredentialsJSON([]byte(t.cfg.CredsJson)))
	}

	cl, err := googleTranslate.NewTranslationClient(ctx, clOpts...)
	if err != nil {
		return nil, fmt.Errorf("Failed to create google translate client: %w", err)
	}
	defer cl.Close()

	config := &googleTranslatePb.TranslateTextRequest{
		Parent:             "projects/" + t.cfg.ProjectId + "/locations/global",
		SourceLanguageCode: r.From,
		TargetLanguageCode: r.To,
		MimeType:           "text/plain",
		Contents:           r.Text,
	}

	resp, err := cl.TranslateText(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("Tranlsation failed: %w", err)
	}

	var result []string
	for _, t := range resp.GetTranslations() {
		result = append(result, t.GetTranslatedText())
	}

	item := &TranlsationResponce{
		Title:       result[0],
		Description: result[1],
		Link: fmt.Sprintf(
			"https://translate.google.com/translate?sl=%s&tl=%s&u=%s",
			r.From, r.To, r.Link,
		),
	}
	return item, nil
}
