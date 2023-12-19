package structs

import "strings"

type TranslatorsConfig struct {
	GoogleCloud interface{} `yaml:"google_cloud"`
}

type FeedConfig struct {
	Source     string                `yaml:"source"`
	Category   string                `yaml:"category"`
	URL        string                `yaml:"url"`
	Language   string                `yaml:"language"`
	ItemsLimit int                   `yaml:"items_limit"`
	Notify     []FeedNotifyConfig    `yaml:"notify"`
	Translates []FeedTranslateConfig `yaml:"translates"`
}

func (f FeedConfig) GetId() string {
	return strings.ToLower(strings.ReplaceAll(f.Source, " ", "_") + "." + strings.ReplaceAll(f.Category, " ", "_"))
}

type FeedTranslateConfig struct {
	To string `yaml:"to"`
}

type FeedNotifyConfig struct {
	Type   string `yaml:"type"`
	ChatId int64  `yaml:"chat_id"`
}

type FeedItem struct {
	Title       string
	Description string
	Link        string
	Source      string
}
