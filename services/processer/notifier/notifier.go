package notifier

import (
	"broadcaster/structs"
	"context"
)

type Notifier interface {
	Notify(ctx context.Context, r NotificationRequest) error
	NewRequest(fn structs.RssFeedNotification, item *structs.RssFeedItem) NotificationRequest
}

type NotificationRequest struct {
	Source  string
	To      []string
	Message string
}
