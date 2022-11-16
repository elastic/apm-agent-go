// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package apmstreadwayamqp // import "go.elastic.co/apm/module/apmstreadwayamqp"

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

// WrapChannel wraps ampq.Channel and returns
// apmstreadwayamqp.WrappedChannel which wraps amqp.Channel
// in a traced manner
func WrapChannel(ch *amqp.Channel) WrappedChannel {
	return WrappedChannel{Channel: ch, ctx: context.Background()}
}

// WithContext supplies context.Context to apmstreadwayamqp.WrappedChannel.
func (c WrappedChannel) WithContext(ctx context.Context) WrappedChannel {
	return WrappedChannel{Channel: c.Channel, ctx: ctx}
}

// Publish publishes a message and returns an error if encountered.
//
// Publish will trace the operation as a span if the context associated with the channel
// (i.e. supplied with WithContext) contains an `*apm.Transaction.`. The trace context
// will be propagated as headers in the published message.
func (c WrappedChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	ctx := c.ctx
	var sn string
	if len(exchange) == 0 {
		sn = "<default>"
	} else {
		sn = exchange
	}
	tx := apm.TransactionFromContext(ctx)
	if tx == nil {
		return c.Channel.Publish(exchange, key, mandatory, immediate, msg)
	}

	traceContext := tx.TraceContext()
	if traceContext.Options.Recorded() {
		span, _ := apm.StartSpanOptions(ctx, sn, "messaging", apm.SpanOptions{ExitSpan: true})
		if !span.Dropped() {
			traceContext = span.TraceContext()
			span.Subtype = "rabbitmq"
			defer span.End()
		} else {
			span.End()
		}
	}
	InjectTraceContext(traceContext, msg)

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
