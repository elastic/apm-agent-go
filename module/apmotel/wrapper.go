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

	"go.elastic.co/apm/v2"
	"go.opentelemetry.io/otel/trace"
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
	ctx = oldOverrideContextWithSpan(ctx, apmSpan)
	return trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID(apmSpan.TraceContext().Trace),
		SpanID:     trace.SpanID(apmSpan.TraceContext().Span),
		TraceFlags: trace.TraceFlags(0).WithSampled(!apmSpan.Dropped()),
	}))
}

func contextWithTransaction(ctx context.Context, apmTransaction *apm.Transaction) context.Context {
	ctx = oldOverrideContextWithTransaction(ctx, apmTransaction)
	return trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID(apmTransaction.TraceContext().Trace),
		SpanID:     trace.SpanID(apmTransaction.TraceContext().Span),
		TraceFlags: trace.TraceFlags(0).WithSampled(apmTransaction.Sampled()),
	}))
}
