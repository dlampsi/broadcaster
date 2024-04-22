package translator

import (
	"context"
)

// Validates interface compliance
var _ Translator = (*MockTranslator)(nil)

// Mock translator doing nothing, just returns original text.
type MockTranslator struct{}

func NewMockTranslator() *MockTranslator {
	return &MockTranslator{}
}

func (t *MockTranslator) Translate(ctx context.Context, r TranlsationRequest) (*TranlsationResponce, error) {
	return &TranlsationResponce{
		Title:       r.Text[0],
		Description: r.Text[1],
		Link:        r.Link,
	}, nil
}
