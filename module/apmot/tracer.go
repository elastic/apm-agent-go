package apmot

import (
	"io"
	"net/http"
	"net/textproto"

	"github.com/opentracing/opentracing-go"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/module/apmhttp"
)

// New returns a new opentracing.Tracer backed by the supplied
// Elastic APM tracer.
//
// By default, the returned tracer will use elasticapm.DefaultTracer.
// This can be overridden by using a WithTracer option.
func New(opts ...Option) opentracing.Tracer {
	t := &otTracer{tracer: elasticapm.DefaultTracer}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// otTracer is an opentracing.Tracer backed by an elasticapm.Tracer.
type otTracer struct {
	tracer *elasticapm.Tracer
}

// StartSpan starts a new OpenTracing span with the given name and zero or more options.
func (t *otTracer) StartSpan(name string, opts ...opentracing.StartSpanOption) opentracing.Span {
	sso := opentracing.StartSpanOptions{}
	for _, o := range opts {
		o.Apply(&sso)
	}
	return t.StartSpanWithOptions(name, sso)
}

// StartSpanWithOptions starts a new OpenTracing span with the given
// name and options.
func (t *otTracer) StartSpanWithOptions(name string, opts opentracing.StartSpanOptions) opentracing.Span {
	ctx := spanContext{tracer: t}
	parentCtx, _ := parentSpanContext(opts.References)
	if parentCtx.tracer == t && parentCtx.tx != nil {
		// tx is non-nil, which means the parent is a process-local
		// transaction or span. Create a sub-span.
		ctx.tx = parentCtx.tx
		ctx.span = ctx.tx.StartSpan(name, "", parentCtx.span)
		if !opts.StartTime.IsZero() {
			ctx.span.Timestamp = opts.StartTime
		}
		ctx.traceContext = ctx.span.TraceContext()
	} else {
		// tx is nil or comes from another tracer, so we must create
		// a new transaction. It's possible that we have a non-local
		// parent transaction, so pass in the (possibly-zero) trace
		// context.
		ctx.tx = t.tracer.StartTransactionOptions(name, "", elasticapm.TransactionOptions{
			TraceContext: parentCtx.traceContext,
			Start:        opts.StartTime,
		})
		ctx.traceContext = ctx.tx.TraceContext()
	}
	// Because the Context method can be called at any time after
	// the span is finished, we cannot pool the objects.
	return &otSpan{
		tracer: t,
		tx:     ctx.tx,
		span:   ctx.span,
		tags:   opts.Tags,
		ctx:    ctx,
	}
}

func (t *otTracer) Inject(sc opentracing.SpanContext, format interface{}, carrier interface{}) error {
	spanContext, ok := sc.(spanContext)
	if !ok {
		return opentracing.ErrInvalidSpanContext
	}
	switch format {
	case opentracing.TextMap, opentracing.HTTPHeaders:
		writer, ok := carrier.(opentracing.TextMapWriter)
		if !ok {
			return opentracing.ErrInvalidCarrier
		}
		headerValue := apmhttp.FormatTraceparentHeader(spanContext.traceContext)
		writer.Set(apmhttp.TraceparentHeader, headerValue)
		return nil
	case opentracing.Binary:
		writer, ok := carrier.(io.Writer)
		if !ok {
			return opentracing.ErrInvalidCarrier
		}
		return binaryInject(writer, spanContext.traceContext)
	default:
		return opentracing.ErrUnsupportedFormat
	}
}

func (t *otTracer) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	switch format {
	case opentracing.TextMap, opentracing.HTTPHeaders:
		var headerValue string
		switch carrier := carrier.(type) {
		case opentracing.HTTPHeadersCarrier:
			headerValue = http.Header(carrier).Get(apmhttp.TraceparentHeader)
		case opentracing.TextMapReader:
			carrier.ForeachKey(func(key, val string) error {
				if textproto.CanonicalMIMEHeaderKey(key) == apmhttp.TraceparentHeader {
					headerValue = val
					return io.EOF // arbitrary error to break loop
				}
				return nil
			})
		default:
			return nil, opentracing.ErrInvalidCarrier
		}
		if headerValue == "" {
			return nil, opentracing.ErrSpanContextNotFound
		}
		traceContext, err := apmhttp.ParseTraceparentHeader(headerValue)
		if err != nil {
			return nil, err
		}
		return spanContext{tracer: t, traceContext: traceContext}, nil
	case opentracing.Binary:
		reader, ok := carrier.(io.Reader)
		if !ok {
			return nil, opentracing.ErrInvalidCarrier
		}
		traceContext, err := binaryExtract(reader)
		if err != nil {
			return nil, err
		}
		return spanContext{tracer: t, traceContext: traceContext}, nil
	default:
		return nil, opentracing.ErrUnsupportedFormat
	}
}

// Option sets options for the OpenTracing Tracer implementation.
type Option func(*otTracer)

// WithTracer returns an Option which sets t as the underlying
// elasticapm.Tracer for constructing an OpenTracing Tracer.
func WithTracer(t *elasticapm.Tracer) Option {
	if t == nil {
		panic("t == nil")
	}
	return func(o *otTracer) {
		o.tracer = t
	}
}

// TODO(axw) handle binary format once Trace-Context defines one.
// OpenTracing mandates that all implementations "MUST" support all
// of the builtin formats.

var (
	binaryInject  = binaryInjectUnsupported
	binaryExtract = binaryExtractUnsupported
)

func binaryInjectUnsupported(w io.Writer, traceContext elasticapm.TraceContext) error {
	return opentracing.ErrUnsupportedFormat
}

func binaryExtractUnsupported(r io.Reader) (elasticapm.TraceContext, error) {
	return elasticapm.TraceContext{}, opentracing.ErrUnsupportedFormat
}
