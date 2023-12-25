package service

import (
	"context"

	"go.uber.org/zap"
)

var _ Notifier = (*MockNotifier)(nil)

type MockNotifier struct {
	logger *zap.SugaredLogger
}

func NewMockNotifier(logger *zap.SugaredLogger) *MockNotifier {
	return &MockNotifier{
		logger: logger,
	}
}

func (m *MockNotifier) Notify(ctx context.Context, r NotificationRequest) error {
	m.logger.With("to", r.To, "message", r.Message).Debug("MOCK")
	return nil
}
