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

//go:build go1.18
// +build go1.18

package apmotel // import "go.elastic.co/apm/module/apmotel/v2"

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"

	"go.elastic.co/apm/v2"
)

type tracer struct {
	provider *tracerProvider
}

func newTracer(p *tracerProvider) *tracer {
	return &tracer{p}
}

// Start forwards the call to APM Agent
func (t *tracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	config := trace.NewSpanStartConfig(opts...)

	startTime := config.Timestamp()
	if startTime.IsZero() {
		startTime = time.Now()
	}

	s := &span{
		provider:   t.provider,
		attributes: config.Attributes(),
		spanKind:   config.SpanKind(),
		startTime:  startTime,
	}

	var psc trace.SpanContext
	if config.NewRoot() {
		ctx = trace.ContextWithSpanContext(ctx, psc)
	} else {
		// If not root, check if a *span is already present, if it's not,
		// attempt to obtain an APM transaction from the context.
		psc = trace.SpanContextFromContext(ctx)
		s.spanContext = psc

		var parent *span
		var ok bool
		if parent, ok = trace.SpanFromContext(ctx).(*span); !ok {
			// Try to find a obtain an APM transaction from the agent context.
			if tx := apm.TransactionFromContext(ctx); tx != nil {
				parent = &span{tx: tx}
			}
		}

		// Use the parent if it exists. Otherwise, we'll create a new
		// transaction using the trace context from `psc`.
		if parent != nil {
			// This is a child span. Create a span, not a transaction.
			// The parent may be a span or a transaction.
			var tc apm.TraceContext
			if parent.span != nil {
				tc = parent.span.TraceContext()
			} else {
				tc = parent.tx.TraceContext()
			}
			s.span = parent.tx.StartSpanOptions(spanName, "", apm.SpanOptions{
				Parent: tc,
				Start:  startTime,
			})
			ctx = apm.ContextWithSpan(ctx, s.span)
			s.tx = parent.tx
			s.spanContext = trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    trace.TraceID(s.span.TraceContext().Trace),
				SpanID:     trace.SpanID(s.span.TraceContext().Span),
				TraceFlags: trace.TraceFlags(0).WithSampled(s.tx.Sampled()),
			})
			return trace.ContextWithSpan(ctx, s), s
		}
	}

	var tranOpts apm.TransactionOptions
	if psc.HasTraceID() && psc.HasSpanID() {
		tranOpts.TraceContext = apm.TraceContext{
			Trace:   [16]byte(psc.TraceID()),
			Span:    [8]byte(psc.SpanID()),
			Options: apm.TraceOptions(0).WithRecorded(psc.IsSampled()),
		}
	}
	s.tx = t.provider.apmTracer.StartTransactionOptions(spanName, "", tranOpts)
	ctx = apm.ContextWithTransaction(ctx, s.tx)
	s.spanContext = trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID(s.tx.TraceContext().Trace),
		SpanID:     trace.SpanID(s.tx.TraceContext().Span),
		TraceFlags: trace.TraceFlags(0).WithSampled(s.tx.Sampled()),
	})

	return trace.ContextWithSpan(ctx, s), s
}
