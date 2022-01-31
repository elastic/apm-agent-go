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

package apmotel // import "go.elastic.co/apm/module/apmotel/v2"

import (
	"context"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"go.elastic.co/apm/v2"
)

// Tracer creates a named tracer that implements otel.Tracer.
// If the name is an empty string then provider uses default name.
func Tracer(name string, opts ...trace.TracerOption) *tracer {
	if name == "" {
		name = "otel_bridge"
	}

	apmOpts := apm.TracerOptions{ServiceName: name}
	if version := trace.NewTracerConfig(opts).InstrumentationVersion(); version != "" {
		apmOpts.ServiceVersion = version
	}

	t := apm.NewTracerOptions(apmOpts)

	return &tracer{inner: t}
}

// SetTracerProvider is a noop function to match opentelemetry's trace module.
func SetTracerProvider(_ trace.TracerProvider) {}

// GetTracerProvider returns a trace.TraceProvider that executes this package's
// Trace() function.
func GetTracerProvider() trace.TracerProvider {
	return tracerFunc(Tracer)
}

type tracerFunc func(string, ...trace.TracerOptions) trace.Tracer

func (fn tracerFunc) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return fn(name, opts)
}

type tracer struct {
	inner apm.Tracer
}

// Start starts a new trace.Span. The span is stored in the returned context.
func (t *tracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return newSpan(ctx, t, spanName, opts)
}

func newSpan(ctx context.Context, t *tracer, name string, opts ...trace.SpanStartOption) (ctx.Context, *span) {
	cfg := trace.NewSpanStartConfig(opts)

	var rpc, http bool
	for attr := range cfg.Attributes() {
		if attr.Value.Type() == attribute.INVALID {
			continue
		}
		switch attr.Name {
		case "rpc.system":
			rpc = true
		case "http.url", "http.scheme":
			http = true
		}
	}

	txType := "unknown"
	switch cfg.SpanKind() {
	case trace.SpanKindServer:
		if rpc || http {
			txType = "request"
		}
	case trace.SpanKindConsumer:
		// TODO: How do we define isMessaging?
		if isMessaging {
			txType = "messaging"
		}
	}

	// Question: This is lowercase, but the example in our docs is showing them as being uppercase
	// Using uppercase for now.
	// https://github.com/open-telemetry/opentelemetry-go/blob/trace/v1.3.0/trace/trace.go#L469-L484
	// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/api.md#spankind
	spanKind := strings.ToUpper(cfg.SpanKind().String())
	spanCtx := trace.SpanContextFromContext(ctx)
	tx := apm.TransactionFromContext(ctx)

	if cfg.NewRoot() {
		return newRootTransaction(ctx, t, spanCtx, cfg.Attributes(), spanKind, name, txType)
	} else if spanCtx.IsValid() {
		txCtx := apm.TraceContext{
			TraceOptions: spanCtx.TraceFlags(),
		}
		if spanCtx.HasTraceID() {
			txCtx.Trace = spanCtx.TraceID()
		}
		if spanCtx.HasSpanID() {
			txCtx.Span = spanCtx.SpanID()
		}
		txOpts := apm.TransactionOptions{TraceContext: txCtx}

		// If timestamp has been set, ie. it's not the default value,
		// set it on txOpts.
		if start := cfg.Timestamp(); start.After(time.Unix(0, 0)) {
			txOpts.Start = start
		}

		tx := t.inner.StartTransactionOptions(name, txType, txOpts)
		// TODO: Check that apm-server version to decide how to set span_kind.
		tx.Context.SetLabel("span_kind", spanKind)
		for attr := range cfg.Attributes() {
			tx.Context.SetLabel(attr.Key, attr.Value)
		}
		ctx := apm.ContextWithTransaction(ctx, tx)
		return ctx, &transaction{inner: tx, spanCtx: spanCtx, tracer: t}
	} else if tx == nil {
		return newRootTransaction(ctx, t, spanCtx, cfg.Attributes(), spanKind, name, txType)
	} else {
		// TODO: Populate data in SpanOptions
		spanOpts := apm.SpanOptions{}
		// If timestamp has been set, ie. it's not the default value,
		// set it on txOpts.
		if start := cfg.Timestamp(); start.After(time.Unix(0, 0)) {
			spanOpts.Start = start
		}

		txID := tx.TraceContext.Span
		s := t.inner.StartSpan(name, spanType, txID, spanOpts)
		// TODO: Check that apm-server version to decide how to set span_kind.
		s.Context.SetLabel("span_kind", spanKind)
		for attr := range cfg.Attributes() {
			tx.Context.SetLabel(attr.Key, attr.Value)
		}
		ctx := apm.ContextWithSpan(ctx, s)
		return ctx, &span{inner: s, tracer: t}
	}
}
