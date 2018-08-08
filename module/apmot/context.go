package apmot

import (
	"github.com/opentracing/opentracing-go"

	"github.com/elastic/apm-agent-go"
)

type spanContext struct {
	// BUG(axw) spanContext must not hold onto any transaction or
	// span objects, as an opentracing.SpanContext may outlive the
	// span to which it relates. To fix this, we need spans at the
	// top level, and a means of creating them from a TraceContext.
	//
	// In most cases this is *probably* OK, because a child-of
	// relation means that the parent span cannot be ended before
	// the child. However, there's no guarantee that the ordering
	// of operations on the OpenTracing API follows the ordering
	// of the events exactly (e.g. you could complete the entire
	// business transaction before emitting events, and then emit
	// events for the transaction and then its spans.)
	tracer *otTracer // for origin of tx/span
	tx     *elasticapm.Transaction
	span   *elasticapm.Span

	traceContext elasticapm.TraceContext
}

// ForeachBaggageItem is a no-op; we do not support baggage propagation.
func (c spanContext) ForeachBaggageItem(handler func(k, v string) bool) {}

func parentSpanContext(refs []opentracing.SpanReference) (spanContext, bool) {
	for _, ref := range refs {
		switch ref.Type {
		case opentracing.ChildOfRef, opentracing.FollowsFromRef:
			return ref.ReferencedContext.(spanContext), true
		}
	}
	return spanContext{}, false
}
