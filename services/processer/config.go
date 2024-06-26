package processer

import "errors"

type TranslationType string

const (
	TranslationTypeMock TranslationType = "mock"
	TranslationTypeGC   TranslationType = "google_cloud"
	TranlsationTypeGAPI TranslationType = "google_api"
)

type Config struct {
	TranslatorType       TranslationType `envconfig:"TRANSLATOR_TYPE" default:"google_api"`
	GoogleCloudProjectId string          `envconfig:"GOOGLE_CLOUD_PROJECT_ID"`
	TelegramBotToken     string          `envconfig:"TELEGRAM_BOT_TOKEN"`
	SlackApiToken        string          `envconfig:"SLACK_API_TOKEN"`
	BackfillHours        int             `envconfig:"BACKFILL_HOURS"`
	MuteNotifications    bool            `envconfig:"MUTE_NOTIFICATIONS"`
	GoogleCloudCreds     string          `envconfig:"GOOGLE_CLOUD_CREDS"`
}

func (c *Config) Validate() error {
	if c.TranslatorType != TranslationTypeMock && c.TranslatorType != TranslationTypeGC && c.TranslatorType != TranlsationTypeGAPI {
		return errors.New("invalid translator type")
	}
	if c.TranslatorType == TranslationTypeGC && c.GoogleCloudProjectId == "" {
		return errors.New("Google Cloud Project ID is required")
	}
	if !c.MuteNotifications && c.TelegramBotToken == "" {
		return errors.New("Telegram Bot Token is required")
	}
	return nil
}
