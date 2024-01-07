package service

import "errors"

type TranslationType string

const (
	TranslationTypeMock TranslationType = "mock"
	TranslationTypeGC   TranslationType = "google_cloud"
)

type Config struct {
	TranslatorType       TranslationType `envconfig:"TRANSLATOR_TYPE" default:"google_cloud"`
	GoogleCloudProjectId string          `envconfig:"GOOGLE_CLOUD_PROJECT_ID"`
	TelegramBotToken     string          `envconfig:"TELEGRAM_BOT_TOKEN"`
	SlackApiToken        string          `envconfig:"SLACK_API_TOKEN"`
	// How many hours back to process
	BackfillHours int `envconfig:"BACKFILL_HOURS"`
	// Do not send notifications
	MuteNotifications bool   `envconfig:"MUTE_NOTIFICATIONS"`
	GoogleCloudCreds  string `envconfig:"GOOGLE_CLOUD_CREDS"`
	StateTTL          int    `envconfig:"STATE_TTL" default:"86400"`
}

func (c *Config) Validate() error {
	if c.TranslatorType != TranslationTypeMock && c.TranslatorType != TranslationTypeGC {
		return errors.New("invalid translator type")
	}
	if c.TranslatorType == TranslationTypeGC && c.GoogleCloudProjectId == "" {
		return errors.New("Google Cloud Project ID is required")
	}
	if c.TelegramBotToken == "" {
		return errors.New("Telegram Bot Token is required")
	}
	return nil
}
