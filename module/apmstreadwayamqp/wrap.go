package apmstreadwayamqp

import (
	"context"
	"github.com/streadway/amqp"
	"go.elastic.co/apm/v2"
)

// WrappedChannel wraps amqp.Channel such that Publish calls are traced,
// and trace context is injected into msg.Headers.
//
// Trace context must be supplied using Channel.WithContext.
// Publish calls ch.Publish.
// NOTE: ctx is not used for cancellation.
type WrappedChannel struct {
	*amqp.Channel
	ctx context.Context
}

func WrapChannel(ch *amqp.Channel) WrappedChannel {
	return WrappedChannel{Channel: ch, ctx: context.Background()}
}

func (c WrappedChannel) WithContext(ctx context.Context) WrappedChannel {
	return WrappedChannel{Channel: c.Channel, ctx: ctx}
}

// Publish publishes a message and returns an error in encountered.
func (c WrappedChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	ctx := c.ctx
	var sn string
	if len(exchange) == 0 {
		sn = "default"
	} else {
		sn = exchange
	}
	span, ctx := apm.StartSpanOptions(ctx, sn, "messaging", apm.SpanOptions{
		ExitSpan: true,
	})
	span.Subtype = "rabbitmq"
	defer span.End()

	InjectTraceContext(span.TraceContext(), msg)

	pubErr := c.Channel.Publish(
		exchange,
		key,
		mandatory,
		immediate,
		msg,
	)

	if pubErr != nil {
		apm.CaptureError(ctx, pubErr).Send()
	}

	return pubErr
}
