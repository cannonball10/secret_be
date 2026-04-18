package notification

import (
	"context"
)

type NotificationConnector interface {
	Send(ctx context.Context, channel *string, message string) error
}

func DefaultNotificationConnector(ctx context.Context) (NotificationConnector, error) {
	return DefaultSlackNotificationConnector(ctx)
}
