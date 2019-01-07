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
		if parentCtx.tracer == t && parentCtx.txSpanContext != nil {
			parentCtx.txSpanContext.mu.RLock()
			defer parentCtx.txSpanContext.mu.RUnlock()
			opts := apm.SpanOptions{
				Parent: parentCtx.traceContext,
				Start:  otSpan.ctx.startTime,
			}
			if parentCtx.txSpanContext.tx != nil {
				otSpan.span = parentCtx.txSpanContext.tx.StartSpanOptions(name, "", opts)
			} else {
				otSpan.span = t.tracer.StartSpan(name, "",
					parentCtx.transactionID,
					opts,
				)
			}
			otSpan.ctx.traceContext = otSpan.span.TraceContext()
			otSpan.ctx.transactionID = parentCtx.transactionID
			otSpan.ctx.txSpanContext = parentCtx.txSpanContext
			return otSpan
		} else if parentCtx.txSpanContext == nil && parentCtx.tx != nil {
			// parentCtx is a synthesized spanContext object. It has no
			// txSpanContext, so we treat it as the transaction creator.
			opts := apm.SpanOptions{
				Parent: parentCtx.traceContext,
				Start:  otSpan.ctx.startTime,
			}
			otSpan.span = parentCtx.tx.StartSpanOptions(name, "", opts)
			otSpan.ctx.traceContext = otSpan.span.TraceContext()
			otSpan.ctx.transactionID = parentCtx.transactionID
			otSpan.ctx.txSpanContext = parentCtx
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
	otSpan.ctx.transactionID = otSpan.ctx.traceContext.Span
	otSpan.ctx.txSpanContext = &otSpan.ctx
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
