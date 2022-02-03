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
func Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	if name == "" {
		name = "otel_bridge"
	}

	apmOpts := apm.TracerOptions{ServiceName: name}
	tracerCfg := trace.NewTracerConfig(opts...)
	if version := (&tracerCfg).InstrumentationVersion(); version != "" {
		apmOpts.ServiceVersion = version
	}

	t, err := apm.NewTracerOptions(apmOpts)
	if err != nil {
		panic(err)
	}

	return &tracer{inner: t}
}

// SetTracerProvider is a noop function to match opentelemetry's trace module.
func SetTracerProvider(_ trace.TracerProvider) {}

// GetTracerProvider returns a trace.TraceProvider that executes this package's
// Trace() function.
func GetTracerProvider() trace.TracerProvider {
	return tracerFunc(Tracer)
}

type tracerFunc func(string, ...trace.TracerOption) trace.Tracer

func (fn tracerFunc) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return fn(name, opts...)
}

type tracer struct {
	inner *apm.Tracer
}

// Start starts a new trace.Span. The span is stored in the returned context.
func (t *tracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return newSpan(ctx, t, spanName, opts...)
}

func newSpan(ctx context.Context, t *tracer, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	cfg := trace.NewSpanStartConfig(opts...)

	var rpc, http, messaging bool
	for _, attr := range cfg.Attributes() {
		if attr.Value.Type() == attribute.INVALID {
			continue
		}
		switch attr.Key {
		case "rpc.system":
			rpc = true
		case "http.url", "http.scheme":
			http = true
		// TODO: Verify we want to set messaging for all of these, and
		// not just `messaging.system`.
		case "messaging.system", "messaging.operation", "message_bus.destination":
			messaging = true
		}
	}

	txType := "unknown"
	switch cfg.SpanKind() {
	case trace.SpanKindServer:
		if rpc || http {
			txType = "request"
		}
	case trace.SpanKindConsumer:
		if messaging {
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
		return newRootTransaction(ctx, t.inner, spanCtx, cfg.Attributes(), spanKind, name, txType)
	} else if spanCtx.IsValid() {
		txCtx := apm.TraceContext{
			Options: apm.TraceOptions(spanCtx.TraceFlags()),
		}
		if spanCtx.HasTraceID() {
			txCtx.Trace = apm.TraceID(spanCtx.TraceID())
		}
		if spanCtx.HasSpanID() {
			txCtx.Span = apm.SpanID(spanCtx.SpanID())
		}
		txOpts := apm.TransactionOptions{TraceContext: txCtx}

		// If timestamp has been set, ie. it's not the default value,
		// set it on txOpts.
		if start := cfg.Timestamp(); start.After(time.Unix(0, 0)) {
			txOpts.Start = start
		}

		tx := t.inner.StartTransactionOptions(name, txType, txOpts)
		tx.Context.SetSpanKind(spanKind)
		tx.Context.SetOtelAttributes(cfg.Attributes()...)
		ctx := apm.ContextWithTransaction(ctx, tx)
		return ctx, &transaction{inner: tx, spanCtx: spanCtx, tracer: t.inner}
	} else if tx == nil {
		return newRootTransaction(ctx, t.inner, spanCtx, cfg.Attributes(), spanKind, name, txType)
	} else {
		// TODO: Populate data in SpanOptions
		spanOpts := apm.SpanOptions{}
		// If timestamp has been set, ie. it's not the default value,
		// set it on txOpts.
		if start := cfg.Timestamp(); start.After(time.Unix(0, 0)) {
			spanOpts.Start = start
		}

		txID := tx.TraceContext().Span
		s := t.inner.StartSpan(name, txType, txID, spanOpts)
		s.Context.SetSpanKind(spanKind)
		tx.Context.SetOtelAttributes(cfg.Attributes()...)
		ctx := apm.ContextWithSpan(ctx, s)
		return ctx, &span{inner: s, tracer: t.inner}
	}
}
