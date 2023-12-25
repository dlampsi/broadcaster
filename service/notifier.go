package service

import "context"

type Notifier interface {
	Notify(ctx context.Context, r NotificationRequest) error
}

type NotificationRequest struct {
	Source  string
	To      []string
	Message string
}
