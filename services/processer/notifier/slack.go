package notifier

import (
	"broadcaster/structs"
	"context"
	"fmt"

	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

type SlackNotifier struct {
	cl     *slack.Client
	logger *zap.SugaredLogger
}

func NewSlackNotifier(token string, logger *zap.SugaredLogger) *SlackNotifier {
	s := &SlackNotifier{
		cl:     slack.New(token),
		logger: logger,
	}
	return s
}

// Implement the Notifier interface
var _ Notifier = (*SlackNotifier)(nil)

func (s *SlackNotifier) Notify(ctx context.Context, r NotificationRequest) error {
	for _, to := range r.To {
		if err := s.notify(ctx, to, r.Source, r.Message); err != nil {
			s.logger.With("err", err.Error()).Errorf("Failed to notify Slack to '%s'", to)
		}
	}
	return nil
}

func (s *SlackNotifier) notify(ctx context.Context, channel, username, msg string) error {
	opts := []slack.MsgOption{
		slack.MsgOptionText(msg, false),
		slack.MsgOptionPostMessageParameters(slack.PostMessageParameters{
			Username: username,
		}),
	}
	_, _, err := s.cl.PostMessage(channel, opts...)
	return err
}

func (s *SlackNotifier) NewRequest(fn structs.RssFeedNotification, item *structs.RssFeedItem) NotificationRequest {
	return NotificationRequest{
		To:     fn.To,
		Source: item.Source,
		Message: fmt.Sprintf(
			"<%s|%s>\n\n%s",
			item.Link,
			item.Title,
			item.Description,
		),
	}
}
