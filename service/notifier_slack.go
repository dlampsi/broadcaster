package service

import (
	"context"

	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

var _ Notifier = (*SlackNotifier)(nil)

type SlackNotifier struct {
	cl     *slack.Client
	logger *zap.SugaredLogger
}

func NewSlackNotifier(token string, logger *zap.SugaredLogger) *SlackNotifier {
	s := &SlackNotifier{
		cl:     slack.New(token),
		logger: logger.Named("slack"),
	}
	return s
}

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
