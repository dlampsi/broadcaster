package structs

type TranslatorsConfig struct {
	GoogleCloud interface{} `yaml:"google_cloud"`
}

type FeedConfig struct {
	Source     string                `yaml:"source"`
	Category   string                `yaml:"category"`
	URL        string                `yaml:"url"`
	ItemsLimit int                   `yaml:"items_limit"`
	Translates []FeedTranslateConfig `yaml:"translates"`
}

type FeedTranslateConfig struct {
	From   string                      `yaml:"from"`
	To     string                      `yaml:"to"`
	Notify []FeedTranslateNotifyConfig `yaml:"notify"`
}

type FeedTranslateNotifyConfig struct {
	Type   string `yaml:"type"`
	ChatId int64  `yaml:"chat_id"`
}

type TranslatedFeedItem struct {
	Title       string
	Description string
	Link        string
	Source      string
}
