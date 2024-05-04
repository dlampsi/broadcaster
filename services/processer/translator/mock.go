package translator

import (
	"context"
	"errors"
)

// Validates interface compliance
var _ Translator = (*MockTranslator)(nil)

// Mock translator doing nothing, just returns original text.
type MockTranslator struct{}

func NewMockTranslator() *MockTranslator {
	return &MockTranslator{}
}

// URL to trigger error in mock translator.
var MockURLForError = "https://fakeurl.local/rss.xml"

func (t *MockTranslator) Translate(ctx context.Context, r TranlsationRequest) (*TranlsationResponce, error) {
	if r.Link == MockURLForError {
		return nil, errors.New("mock error")
	}
	return &TranlsationResponce{
		Title:       r.Text[0],
		Description: r.Text[1],
		Link:        r.Link,
	}, nil
}
