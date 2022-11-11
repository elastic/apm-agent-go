package apmstreadwayamqp

import (
	"context"
	"github.com/streadway/amqp"
	"go.elastic.co/apm/v2"
)

// Extractor is the interface returned by Prepare.
//
// Extractor's Extract method reports incoming messages as transactions.
type Extractor interface {

	// WithContext returns a shallow copy of the Extractor with
	// its context changed to ctx.
	//
	// The reported transactions from incoming messages are
	// put in ctx.
	WithContext(ctx context.Context) Extractor

	// WithTracer returns a shallow copy of the Extractor with
	// its tracer changed to t.
	//
	// Otherwise apm.DefaultTracer() is used.
	WithTracer(t *apm.Tracer) Extractor

	// Extract returns a transaction from the extracted
	// trace information from the headers.
	//
	// The started transaction is stored in the ctx.
	Extract(del amqp.Delivery) (*apm.Transaction, context.Context)
}

// Prepare returns an Extractor which could start apm.Transaction
// related to previous transaction or span, or new one.
func Prepare(headers map[string]interface{}) Extractor {
	return headerExtractor{h: headers, ctx: context.Background(), tr: apm.DefaultTracer()}
}

type headerExtractor struct {
	h   map[string]interface{}
	ctx context.Context
	tr  *apm.Tracer
}

func (h headerExtractor) Extract(del amqp.Delivery) (*apm.Transaction, context.Context) {
	return еxtract(h.ctx, h.tr, del)
}

func (h headerExtractor) WithContext(ctx context.Context) Extractor {
	h.ctx = ctx
	return h
}
func (h headerExtractor) WithTracer(t *apm.Tracer) Extractor {
	h.tr = t
	return h
}

func еxtract(ctx context.Context, tracer *apm.Tracer, del amqp.Delivery) (*apm.Transaction, context.Context) {
	if err := del.Headers.Validate(); err != nil {
		return nil, ctx
	}

	traceContext, ok := getMessageTraceparent(del.Headers, w3cTraceparentHeader)
	if !ok {
		traceContext, ok = getMessageTraceparent(del.Headers, elasticTraceparentHeader)
	}

	if ok {
		traceContext.State, _ = getMessageTracestate(del.Headers, tracestateHeader)
	}

	tx := tracer.StartTransactionOptions("rabbitmq.handle", "messaging", apm.TransactionOptions{TraceContext: traceContext})
	ctx = apm.ContextWithTransaction(ctx, tx)
	return tx, ctx
}
