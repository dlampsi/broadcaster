package translator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

// Validates interface compliance
var _ Translator = (*GoogleApiTranslator)(nil)

type GoogleApiTranslator struct {
	httpCl *http.Client
}

// Creates new free Google API translator.
func NewGoogleApiTranslator() *GoogleApiTranslator {
	httpCl := &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 10 * time.Second,
		},
		Timeout: 10 * time.Second,
	}
	return &GoogleApiTranslator{
		httpCl: httpCl,
	}
}

func (t *GoogleApiTranslator) Translate(ctx context.Context, r TranlsationRequest) (*TranlsationResponce, error) {
	var translates []string
	for _, text := range r.Text {
		ts, err := t.translate(ctx, r.From, r.To, text)
		if err != nil {
			return nil, fmt.Errorf("Failed to translate text: %w", err)
		}
		translates = append(translates, ts)
	}

	if len(translates) != 2 {
		return nil, fmt.Errorf("Unexpected number of translations: %d", len(translates))
	}

	result := &TranlsationResponce{
		Title:       translates[0],
		Description: translates[1],
		Link: fmt.Sprintf(
			"https://translate.google.com/translate?sl=%s&tl=%s&u=%s",
			r.From, r.To, r.Link,
		),
	}

	return result, nil
}

func (t *GoogleApiTranslator) translate(ctx context.Context, from, to, text string) (string, error) {
	uri := fmt.Sprintf(
		"https://translate.googleapis.com/translate_a/single?client=gtx&sl=%s&tl=%s&dt=t&q=%s",
		from, to, url.QueryEscape(text),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", uri, nil)
	if err != nil {
		return "", fmt.Errorf("Failed to create new request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := t.httpCl.Do(req)
	if err != nil {
		return "", fmt.Errorf("Failed to do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Bad response code from Google API (%d): %s", resp.StatusCode, string(body))
	}

	var apiResp []interface{}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("Failed to unmarshal response: %w", err)
	}

	translate, err := parseGoogleApiResp(apiResp)
	if err != nil {
		return "", fmt.Errorf("Failed to parse response: %w", err)
	}

	return translate, nil
}

// Fetches translation strings from teh Google API response.
func parseGoogleApiResp(input []interface{}) (string, error) {
	if len(input) == 0 {
		return "", fmt.Errorf("Empty response from Google API")
	}

	translates, ok := input[0].([]interface{})
	if !ok {
		return "", fmt.Errorf("Failed to get translates from response: %v", input[0])
	}

	var result string

	for _, ts := range translates {
		tss, ok := ts.([]interface{})
		if !ok {
			return "", fmt.Errorf("Failed to parse translate from translates: %v", ts)
		}
		tss0, ok := tss[0].(string)
		if !ok {
			return "", fmt.Errorf("Failed to convert translate to string: %v", tss[0])
		}
		result += tss0
	}

	return result, nil
}
