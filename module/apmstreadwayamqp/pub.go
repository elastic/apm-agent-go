package apmstreadwayamqp

import (
	"context"
	"github.com/streadway/amqp"
	"go.elastic.co/apm/v2"
)

// Publisher is the interface returned by ContextConn.
//
// Publisher's Publish method reports spans using the bound context.
type Publisher interface {

	// WithContext returns a shallow copy of the Publisher with
	// its context changed to ctx.
	//
	// To report publishing as spans, ctx must contain a transaction or span.
	WithContext(ctx context.Context) Publisher

	// Publish publishes a message and returns an error in encountered.
	//
	// The started span is stored in the ctx.
	Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
}

// Wrap wraps amqp.Channel such that its Publish method calls streadway/amqp.Publish.
func Wrap(ch *amqp.Channel) Publisher {
	return msgPublisher{ch: ch, ctx: context.Background()}
}

type msgPublisher struct {
	ch  *amqp.Channel
	ctx context.Context
}

func (c msgPublisher) WithContext(ctx context.Context) Publisher {
	c.ctx = ctx
	return c
}

func (c msgPublisher) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return pub(c.ctx, c.ch, exchange, key, mandatory, immediate, msg)
}

func pub(ctx context.Context, ch *amqp.Channel, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	span, ctx := apm.StartSpanOptions(ctx, "rabbitmq.pub", "messaging", apm.SpanOptions{
		ExitSpan: true,
	})
	span.Subtype = "rabbitmq"
	span.Outcome = "success"
	defer span.End()

	setMessageTraceparent(span.TraceContext(), msg.Headers)
	stateErr := setMessageTracestate(span.TraceContext().State, msg.Headers)
	if stateErr != nil {
		span.Outcome = "failure"
		return stateErr
	}

	pubErr := ch.Publish(
		exchange,
		key,
		false, // mandatory
		false, // immediate
		msg,
	)

	if pubErr != nil {
		span.Outcome = "failure"
	}
	return pubErr
}
