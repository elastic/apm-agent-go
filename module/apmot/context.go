package apmot

import (
	"sync"

	"github.com/opentracing/opentracing-go"

	"github.com/elastic/apm-agent-go"
)

type spanContext struct {
	tracer *otTracer // for origin of tx/span

	txSpanContext *spanContext // spanContext for OT span which created tx
	traceContext  elasticapm.TraceContext
	transactionID elasticapm.SpanID

	mu sync.RWMutex
	tx *elasticapm.Transaction
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
		}
	}
	return nil, false
}
