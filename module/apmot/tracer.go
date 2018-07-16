package apmot

import (
	"github.com/opentracing/opentracing-go"

	"github.com/elastic/apm-agent-go"
)

func init() {
	opentracing.SetGlobalTracer(New(nil))
}

// New returns a new opentracing.Tracer backed by the supplied
// Elastic APM tracer. If the supplied tracer is nil, then
// opentracing.DefaultTracer will be used.
func New(tracer *elasticapm.Tracer) opentracing.Tracer {
	if tracer == nil {
		tracer = elasticapm.DefaultTracer
	}
	return &tracerImpl{tracer: tracer}
}

// Implements the `Tracer` interface.
type tracerImpl struct {
	tracer *elasticapm.Tracer
}

func (t *tracerImpl) StartSpan(
	operationName string,
	opts ...opentracing.StartSpanOption,
) opentracing.Span {
	sso := opentracing.StartSpanOptions{}
	for _, o := range opts {
		o.Apply(&sso)
	}
	return t.StartSpanWithOptions(operationName, sso)
}

func (t *tracerImpl) StartSpanWithOptions(
	operationName string,
	opts opentracing.StartSpanOptions,
) opentracing.Span {

	var ctx SpanContext
	parentCtx, ok := parentSpanContext(opts.References)
	if !ok {
		ctx.tx = t.tracer.StartTransaction(operationName, "")
		if !opts.StartTime.IsZero() {
			ctx.tx.Timestamp = opts.StartTime
		}
	} else {
		if parentCtx.tx != nil {
			ctx.tx = parentCtx.tx
			ctx.span = ctx.tx.StartSpan(operationName, "", parentCtx.span)
			if !opts.StartTime.IsZero() {
				ctx.span.Timestamp = opts.StartTime
			}
		} else {
			// TODO(axw) create a transaction with trace and
			// parent ID taken from the parent context.
			ctx.tx = t.tracer.StartTransaction(operationName, "")
			if !opts.StartTime.IsZero() {
				ctx.tx.Timestamp = opts.StartTime
			}
		}
		if n := len(parentCtx.baggage); n != 0 {
			ctx.baggage = make(map[string]string, n)
			for k, v := range parentCtx.baggage {
				ctx.baggage[k] = v
			}
		}
	}

	// Because the Context method can be called at any time after the span
	// is finished, we cannot pool the objects.
	return &spanImpl{
		tracer: t,
		tx:     ctx.tx,
		span:   ctx.span,
		tags:   opts.Tags,
		ctx:    ctx,
	}
}

func (t *tracerImpl) Inject(sc opentracing.SpanContext, format interface{}, carrier interface{}) error {
	// TODO(axw) propagation.
	return opentracing.ErrUnsupportedFormat
}

func (t *tracerImpl) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	// TODO(axw) propagation.
	return nil, opentracing.ErrUnsupportedFormat
}
