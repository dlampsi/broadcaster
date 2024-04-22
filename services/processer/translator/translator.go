package translator

import (
	"context"
)

type Translator interface {
	Translate(ctx context.Context, r TranlsationRequest) (*TranlsationResponce, error)
}

type TranlsationRequest struct {
	Link string   // Link to the original article
	From string   // Language code of the original article
	To   string   // Language code to translate to
	Text []string // Text to translate
}

type TranlsationResponce struct {
	Title       string
	Description string
	Link        string
}
