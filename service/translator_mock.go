package service

import (
	"a0feed/structs"
	"context"
)

// Validates interface compliance
var _ Translator = (*MockTranslator)(nil)

// Mock translator doing nothing, just returns original text.
type MockTranslator struct{}

func NewMockTranslator() *MockTranslator {
	return &MockTranslator{}
}

func (t *MockTranslator) Translate(ctx context.Context, r TranlsationRequest) (*structs.TranslatedFeedItem, error) {
	return &structs.TranslatedFeedItem{
		Title:       r.Text[0],
		Description: r.Text[1],
		Link:        r.Link,
	}, nil
}
