package apmot

import (
	opentracing "github.com/opentracing/opentracing-go"

	"github.com/elastic/apm-agent-go"
)

// SpanContext holds the basic Span metadata.
type SpanContext struct {
	tx      *elasticapm.Transaction
	span    *elasticapm.Span
	baggage map[string]string
}

// ForeachBaggageItem belongs to the opentracing.SpanContext interface
func (c SpanContext) ForeachBaggageItem(handler func(k, v string) bool) {
	for k, v := range c.baggage {
		if !handler(k, v) {
			break
		}
	}
}

// WithBaggageItem returns an entirely new SpanContext with the
// given key:value baggage pair set.
func (c SpanContext) WithBaggageItem(key, val string) SpanContext {
	var newBaggage map[string]string
	if c.baggage == nil {
		newBaggage = map[string]string{key: val}
	} else {
		newBaggage = make(map[string]string, len(c.baggage)+1)
		for k, v := range c.baggage {
			newBaggage[k] = v
		}
		newBaggage[key] = val
	}
	return SpanContext{tx: c.tx, span: c.span, baggage: newBaggage}
}

func parentSpanContext(refs []opentracing.SpanReference) (SpanContext, bool) {
	for _, ref := range refs {
		switch ref.Type {
		case opentracing.ChildOfRef, opentracing.FollowsFromRef:
			return ref.ReferencedContext.(SpanContext), true
		}
	}
	return SpanContext{}, false
}
