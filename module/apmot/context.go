package apmot

import (
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"

	"go.elastic.co/apm"
)

type spanContext struct {
	tracer *otTracer // for origin of tx/span

	txSpanContext *spanContext // spanContext for OT span which created tx
	traceContext  apm.TraceContext
	transactionID apm.SpanID
	startTime     time.Time

	mu sync.RWMutex
	tx *apm.Transaction
}

// TraceContext returns the trace context for the transaction or span
// associated with this span context.
func (s *spanContext) TraceContext() apm.TraceContext {
	return s.traceContext
}

// Transaction returns the transaction associated with this span context.
func (s *spanContext) Transaction() *apm.Transaction {
	if s.txSpanContext != nil {
		return s.txSpanContext.tx
	}
	return s.tx
}

// ForeachBaggageItem is a no-op; we do not support baggage propagation.
func (*spanContext) ForeachBaggageItem(handler func(k, v string) bool) {}

func parentSpanContext(refs []opentracing.SpanReference) (*spanContext, bool) {
	for _, ref := range refs {
		switch ref.Type {
		case opentracing.ChildOfRef, opentracing.FollowsFromRef:
			if ctx, ok := ref.ReferencedContext.(*spanContext); ok {
				return ctx, ok
			}
			if apmSpanContext, ok := ref.ReferencedContext.(interface {
				Transaction() *apm.Transaction
				TraceContext() apm.TraceContext
			}); ok {
				// The span context is (probably) one of the
				// native Elastic APM span/transaction wrapper
				// types. Synthesize a spanContext so we can
				// automatically correlate the events created
				// through our native API and the OpenTracing API.
				spanContext := &spanContext{
					tx:           apmSpanContext.Transaction(),
					traceContext: apmSpanContext.TraceContext(),
				}
				spanContext.transactionID = spanContext.tx.TraceContext().Span
				return spanContext, true
			}
		}
	}
	return nil, false
}
