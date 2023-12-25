package service

import (
	"context"
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// Implement the Notifier interface
var _ Notifier = (*TelegramNotifier)(nil)

type TelegramNotifier struct {
	bot    *tgbotapi.BotAPI
	logger *zap.SugaredLogger
}

func NewTelegramNotifier(botToken string) (*TelegramNotifier, error) {
	t := &TelegramNotifier{
		logger: zap.NewNop().Sugar(),
	}

	tgbot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, fmt.Errorf("Failed to init a telegram client: %w", err)
	}
	t.bot = tgbot

	return t, nil
}

func (t *TelegramNotifier) Notify(ctx context.Context, r NotificationRequest) error {
	for _, to := range r.To {
		chatId, err := strconv.ParseInt(to, 10, 64)
		if err == nil {
			if err := t.notify(ctx, chatId, r.Message); err != nil {
				return err
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
