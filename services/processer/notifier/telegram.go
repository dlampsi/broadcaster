package notifier

import (
	"broadcaster/structs"
	"context"
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/microcosm-cc/bluemonday"
	"go.uber.org/zap"
)

type TelegramNotifier struct {
	bot    *tgbotapi.BotAPI
	logger *zap.SugaredLogger
}

func NewTelegramNotifier(botToken string, logger *zap.SugaredLogger) (*TelegramNotifier, error) {
	t := &TelegramNotifier{
		logger: logger,
	}

	tgbot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, fmt.Errorf("Failed to init a telegram client: %w", err)
	}
	t.bot = tgbot
	t.logger.Debug("Bot client initialized")

	return t, nil
}

// Implement the Notifier interface
var _ Notifier = (*TelegramNotifier)(nil)

func (t *TelegramNotifier) Notify(ctx context.Context, r NotificationRequest) error {
	for _, to := range r.To {
		chatId, err := strconv.ParseInt(to, 10, 64)
		if err == nil {
			if err := t.notify(ctx, chatId, r.Message); err != nil {
				t.logger.With("err", err.Error()).Errorf("Failed to notify Telegram to '%s'", to)
				t.logger.Debug(r.Message)
			}
		}
	}
	return nil
}

func (t *TelegramNotifier) notify(ctx context.Context, chatId int64, message string) error {
	msg := tgbotapi.NewMessage(chatId, message)
	msg.ParseMode = "markdown"
	msg.DisableWebPagePreview = false

	if _, err := t.bot.Send(msg); err != nil {
		return err
	}
	return nil
}

func (t *TelegramNotifier) NewRequest(fn structs.RssFeedNotification, item *structs.RssFeedItem) NotificationRequest {

	if len(item.Description) > 0 {
		p := bluemonday.StrictPolicy()
		item.Description = p.Sanitize(item.Description)
	}

	return NotificationRequest{
		To: fn.To,
		Message: fmt.Sprintf(
			"*%s* \n\n%s\n\n[%s](%s)",
			item.Title,
			item.Description,
			item.Source,
			item.Link,
		),
	}
}
