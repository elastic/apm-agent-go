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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"go.elastic.co/apm/v2"
)

var (
	oldOverrideContextWithSpan        func(context.Context, *apm.Span) context.Context
	oldOverrideContextWithTransaction func(context.Context, *apm.Transaction) context.Context
)

func init() {
	// We override the apm context functions so that transactions
	// and spans started with the native API are wrapped and made
	// available as OpenTelemetry spans.

	oldOverrideContextWithSpan = apm.OverrideContextWithSpan
	oldOverrideContextWithTransaction = apm.OverrideContextWithTransaction
	apm.OverrideContextWithSpan = contextWithSpan
	apm.OverrideContextWithTransaction = contextWithTransaction
}

func contextWithSpan(ctx context.Context, apmSpan *apm.Span) context.Context {
	var provider *tracerProvider
	if p, ok := otel.GetTracerProvider().(*tracerProvider); ok {
		provider = p
	}

	ctx = oldOverrideContextWithSpan(ctx, apmSpan)

	return trace.ContextWithSpan(ctx, &span{
		provider: provider,

		spanContext: trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    trace.TraceID(apmSpan.TraceContext().Trace),
			SpanID:     trace.SpanID(apmSpan.TraceContext().Span),
			TraceFlags: trace.TraceFlags(0).WithSampled(!apmSpan.Dropped()),
		}),

		span: apmSpan,
	})
}

func contextWithTransaction(ctx context.Context, apmTransaction *apm.Transaction) context.Context {
	var provider *tracerProvider
	if p, ok := otel.GetTracerProvider().(*tracerProvider); ok {
		provider = p
	}
	ctx = oldOverrideContextWithTransaction(ctx, apmTransaction)

	return trace.ContextWithSpan(ctx, &span{
		provider: provider,

		spanContext: trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    trace.TraceID(apmTransaction.TraceContext().Trace),
			SpanID:     trace.SpanID(apmTransaction.TraceContext().Span),
			TraceFlags: trace.TraceFlags(0).WithSampled(apmTransaction.Sampled()),
		}),

		tx: apmTransaction,
	})
}
