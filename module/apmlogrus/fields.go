package apmlogrus

import (
	"context"

	"github.com/sirupsen/logrus"

	"go.elastic.co/apm"
)

const (
	// FieldKeyTraceID is the field key for the trace ID.
	FieldKeyTraceID = "trace.id"

	// FieldKeyTransactionID is the field key for the transaction ID.
	FieldKeyTransactionID = "transaction.id"

	// FieldKeySpanID is the field key for the span ID.
	FieldKeySpanID = "span.id"
)

// TraceContext returns a logrus.Fields containing the trace
// context of the transaction and span contained in ctx, if any.
func TraceContext(ctx context.Context) logrus.Fields {
	tx := apm.TransactionFromContext(ctx)
	if tx == nil {
		return nil
	}
	traceContext := tx.TraceContext()
	fields := logrus.Fields{
		FieldKeyTraceID:       traceContext.Trace,
		FieldKeyTransactionID: traceContext.Span,
	}
	if span := apm.SpanFromContext(ctx); span != nil {
		fields[FieldKeySpanID] = span.TraceContext().Span
	}
	return fields
}
