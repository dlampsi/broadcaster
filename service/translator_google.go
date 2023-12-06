package service

import (
	"broadcaster/structs"
	"context"
	"fmt"

	googleTranslate "cloud.google.com/go/translate/apiv3"
	googleTranslatePb "cloud.google.com/go/translate/apiv3/translatepb"
)

// Validates interface compliance
var _ Translator = (*GoogleCloudTranslator)(nil)

type GoogleCloudTranslator struct {
	projectId string
}

func NewGoogleCloudTranslator(projectId string) *GoogleCloudTranslator {
	return &GoogleCloudTranslator{
		projectId: projectId,
	}
}

func (t *GoogleCloudTranslator) Translate(ctx context.Context, r TranlsationRequest) (*structs.TranslatedFeedItem, error) {
	cl, err := googleTranslate.NewTranslationClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to create google translate client: %w", err)
	}
	defer cl.Close()

	config := &googleTranslatePb.TranslateTextRequest{
		Parent:             "projects/" + t.projectId + "/locations/global",
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

	item := &structs.TranslatedFeedItem{
		Title:       result[0],
		Description: result[1],
		Link: fmt.Sprintf(
			"https://translate.google.com/translate?sl=%s&tl=%s&u=%s",
			r.From, r.To, r.Link,
		),
	}
	return item, nil
}
