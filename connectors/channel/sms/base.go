package sms

import "context"

// SMSConnector is an abstract SMS interface that can be extended for concrete
// implementations like Twilio, Sendblue, etc.
type SMSConnector interface {
	Send(ctx context.Context, to, from, message string) error
}

func DefaultSMSConnector(ctx context.Context) (SMSConnector, error) {
	return DefaultTwilioSMSConnector(ctx, nil)
}
