package apmot

import (
	"time"

	opentracing "github.com/opentracing/opentracing-go"

	"github.com/elastic/apm-agent-go"
)

func init() {
	opentracing.SetGlobalTracer(New(elasticapm.DefaultTracer))
}

func New(tracer *elasticapm.Tracer) opentracing.Tracer {
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

	startTime := opts.StartTime
	if startTime.IsZero() {
		startTime = time.Now()
	}

	var ctx SpanContext
	parentCtx, ok := parentSpanContext(opts.References)
	if !ok {
		ctx.tx = t.tracer.StartTransaction(operationName, "")
	} else {
		ctx.tx = parentCtx.tx
		ctx.span = ctx.tx.StartSpan(operationName, "", parentCtx.span)
		if n := len(parentCtx.baggage); n != 0 {
			ctx.baggage = make(map[string]string, n)
			for k, v := range parentCtx.baggage {
				ctx.baggage[k] = v
			}
		}
	}

	// TODO(axw) pool spanImpls to avoid allocation overhead.
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
