package notification

import (
	"context"
	"log/slog"
	"os"

	"github.com/slack-go/slack"
)

type SlackNotification struct {
	Client *slack.Client
}

func NewSlackNotification(client *slack.Client) *SlackNotification {
	return &SlackNotification{Client: client}
}

func DefaultSlackNotificationConnector(ctx context.Context, options ...slack.Option) (*SlackNotification, error) {
	client := slack.New(os.Getenv("SLACK_API_TOKEN"), options...)
	return NewSlackNotification(client), nil
}

func (s *SlackNotification) Send(ctx context.Context, channel *string, message string) error {
	_, _, _, err := s.Client.SendMessageContext(ctx, *channel, slack.MsgOptionText(message, false))
	if err != nil {
		slog.ErrorContext(ctx, "slack Send failed", "channel", *channel, "error", err)
	}
	return err
}
