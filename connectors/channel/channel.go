package channel

import (
	"context"

	"github.com/cannonball10/foundation/connectors/channel/sms"
)

type ChannelConnector struct {
	SMS sms.SMSConnector
}

func NewChannelConnector(sms sms.SMSConnector) ChannelConnector {
	return ChannelConnector{
		SMS: sms,
	}
}

func DefaultChannelConnector(ctx context.Context) ChannelConnector {
	connector, err := sms.DefaultSMSConnector(ctx)
	if err != nil {
		panic(err)
	}
	return NewChannelConnector(connector)
}
