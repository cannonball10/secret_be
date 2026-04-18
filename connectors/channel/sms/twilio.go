package sms

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

type TwilioSMSConnector struct {
	client *twilio.RestClient
}

func NewTwilioSMSConnector(client *twilio.RestClient) *TwilioSMSConnector {
	return &TwilioSMSConnector{client: client}
}

func DefaultTwilioSMSConnector(ctx context.Context, subAccountSid *string) (*TwilioSMSConnector, error) {
	if subAccountSid == nil {
		subsid := ""
		subAccountSid = &subsid
	}
	accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	if accountSid == "" || authToken == "" {
		return nil, errors.New("TWILIO_ACCOUNT_SID and TWILIO_AUTH_TOKEN must be set")
	}
	return NewTwilioSMSConnector(twilio.NewRestClientWithParams(twilio.ClientParams{
		Username:   accountSid,
		Password:   authToken,
		AccountSid: *subAccountSid,
	})), nil
}

func (c *TwilioSMSConnector) Send(ctx context.Context, to, from, message string) error {
	_, err := c.client.Api.CreateMessage(&openapi.CreateMessageParams{
		To:   &to,
		From: &from,
		Body: &message,
	})
	if err != nil {
		slog.ErrorContext(ctx, "sms send failed", "error", err)
	}
	return err
}
