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

package apmot

import (
	"io"
	"net/http"
	"net/textproto"
	"time"

	opentracing "github.com/opentracing/opentracing-go"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

// New returns a new opentracing.Tracer backed by the supplied
// Elastic APM tracer.
//
// By default, the returned tracer will use apm.DefaultTracer.
// This can be overridden by using a WithTracer option.
func New(opts ...Option) opentracing.Tracer {
	t := &otTracer{tracer: apm.DefaultTracer}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// otTracer is an opentracing.Tracer backed by an apm.Tracer.
type otTracer struct {
	tracer *apm.Tracer
}

// StartSpan starts a new OpenTracing span with the given name and zero or more options.
func (t *otTracer) StartSpan(name string, opts ...opentracing.StartSpanOption) opentracing.Span {
	sso := opentracing.StartSpanOptions{}
	for _, o := range opts {
		o.Apply(&sso)
	}
	return t.StartSpanWithOptions(name, sso)
}

// StartSpanWithOptions starts a new OpenTracing span with the given name and options.
func (t *otTracer) StartSpanWithOptions(name string, opts opentracing.StartSpanOptions) opentracing.Span {
	// Because the Context method can be called at any time after
	// the span is finished, we cannot pool the objects.
	otSpan := &otSpan{
		tracer: t,
		tags:   opts.Tags,
		ctx: spanContext{
			tracer:    t,
			startTime: opts.StartTime,
		},
	}
	if opts.StartTime.IsZero() {
		otSpan.ctx.startTime = time.Now()
	}

	var parentTraceContext apm.TraceContext
	if parentCtx, ok := parentSpanContext(opts.References); ok {
		if parentCtx.tx != nil && (parentCtx.tracer == t || parentCtx.tracer == nil) {
			opts := apm.SpanOptions{
				Parent: parentCtx.traceContext, // parent span
				Start:  otSpan.ctx.startTime,
			}
			otSpan.span = parentCtx.tx.StartSpanOptions(name, "", opts)
			otSpan.ctx.tx = parentCtx.tx
			otSpan.ctx.traceContext = otSpan.span.TraceContext()
			return otSpan
		}
		parentTraceContext = parentCtx.traceContext
	}

	// There's no local parent context created by this tracer.
	otSpan.ctx.tx = t.tracer.StartTransactionOptions(name, "", apm.TransactionOptions{
		TraceContext: parentTraceContext,
		Start:        otSpan.ctx.startTime,
	})
	otSpan.ctx.traceContext = otSpan.ctx.tx.TraceContext()
	return otSpan
}

func (t *otTracer) Inject(sc opentracing.SpanContext, format interface{}, carrier interface{}) error {
	spanContext, ok := sc.(*spanContext)
	if !ok {
		return opentracing.ErrInvalidSpanContext
	}
	switch format {
	case opentracing.TextMap, opentracing.HTTPHeaders:
		writer, ok := carrier.(opentracing.TextMapWriter)
		if !ok {
			return opentracing.ErrInvalidCarrier
		}
		tx := spanContext.Transaction()
		headerValue := apmhttp.FormatTraceparentHeader(spanContext.traceContext)
		writer.Set(apmhttp.W3CTraceparentHeader, headerValue)
		if tx.ShouldPropagateLegacyHeader() {
			writer.Set(apmhttp.ElasticTraceparentHeader, headerValue)
		}
		if tracestate := spanContext.traceContext.State.String(); tracestate != "" {
			writer.Set(apmhttp.TracestateHeader, tracestate)
		}
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
		var traceparentHeaderValue string
		var tracestateHeaderValues []string
		switch carrier := carrier.(type) {
		case opentracing.HTTPHeadersCarrier:
			traceparentHeaderValue = http.Header(carrier).Get(apmhttp.ElasticTraceparentHeader)
			if traceparentHeaderValue == "" {
				traceparentHeaderValue = http.Header(carrier).Get(apmhttp.W3CTraceparentHeader)
			}
			tracestateHeaderValues = http.Header(carrier)[apmhttp.TracestateHeader]
		case opentracing.TextMapReader:
			carrier.ForeachKey(func(key, val string) error {
				switch textproto.CanonicalMIMEHeaderKey(key) {
				case apmhttp.ElasticTraceparentHeader:
					traceparentHeaderValue = val
				case apmhttp.W3CTraceparentHeader:
					// The Elastic header value always trumps the W3C one,
					// to ensure backwards compatibility, hence we only set
					// the value if not set already.
					if traceparentHeaderValue == "" {
						traceparentHeaderValue = val
					}
				case apmhttp.TracestateHeader:
					tracestateHeaderValues = append(tracestateHeaderValues, val)
				}
				return nil
			})
		default:
			return nil, opentracing.ErrInvalidCarrier
		}
		if traceparentHeaderValue == "" {
			return nil, opentracing.ErrSpanContextNotFound
		}
		traceContext, err := apmhttp.ParseTraceparentHeader(traceparentHeaderValue)
		if err != nil {
			return nil, err
		}
		traceContext.State, _ = apmhttp.ParseTracestateHeader(tracestateHeaderValues...)
		return &spanContext{tracer: t, traceContext: traceContext}, nil
	case opentracing.Binary:
		reader, ok := carrier.(io.Reader)
		if !ok {
			return nil, opentracing.ErrInvalidCarrier
		}
		traceContext, err := binaryExtract(reader)
		if err != nil {
			return nil, err
		}
		return &spanContext{tracer: t, traceContext: traceContext}, nil
	default:
		return nil, opentracing.ErrUnsupportedFormat
	}
}

// Option sets options for the OpenTracing Tracer implementation.
type Option func(*otTracer)

// WithTracer returns an Option which sets t as the underlying
// apm.Tracer for constructing an OpenTracing Tracer.
func WithTracer(t *apm.Tracer) Option {
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

func binaryInjectUnsupported(w io.Writer, traceContext apm.TraceContext) error {
	return opentracing.ErrUnsupportedFormat
}

func binaryExtractUnsupported(r io.Reader) (apm.TraceContext, error) {
	return apm.TraceContext{}, opentracing.ErrUnsupportedFormat
}
