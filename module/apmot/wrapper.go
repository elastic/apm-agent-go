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
	"context"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"go.elastic.co/apm"
	"go.elastic.co/apm/internal/apmcontext"
)

func init() {
	// We override the apmcontext functions so that transactions
	// and spans started with the native API are wrapped and made
	// available as OpenTracing spans.
	apmcontext.ContextWithSpan = contextWithSpan
	apmcontext.ContextWithTransaction = contextWithTransaction
	apmcontext.SpanFromContext = spanFromContext
	apmcontext.TransactionFromContext = transactionFromContext
}

func contextWithSpan(ctx context.Context, apmSpan interface{}) context.Context {
	tx, _ := transactionFromContext(ctx).(*apm.Transaction)
	return opentracing.ContextWithSpan(ctx, apmSpanWrapper{
		spanContext: apmSpanWrapperContext{
			span:        apmSpan.(*apm.Span),
			transaction: tx,
		},
	})
}

func contextWithTransaction(ctx context.Context, apmTransaction interface{}) context.Context {
	return opentracing.ContextWithSpan(ctx, apmTransactionWrapper{
		spanContext: apmTransactionWrapperContext{
			transaction: apmTransaction.(*apm.Transaction),
		},
	})
}

func spanFromContext(ctx context.Context) interface{} {
	otSpan, _ := opentracing.SpanFromContext(ctx).(interface {
		Span() *apm.Span
	})
	if otSpan == nil {
		return nil
	}
	return otSpan.Span()
}

func transactionFromContext(ctx context.Context) interface{} {
	otSpan := opentracing.SpanFromContext(ctx)
	if otSpan == nil {
		return nil
	}
	if apmSpanContext, ok := otSpan.Context().(interface {
		Transaction() *apm.Transaction
	}); ok {
		return apmSpanContext.Transaction()
	}
	return nil
}

// apmSpanWrapperContext is an opentracing.SpanContext that wraps
// an apm.Span and apm.Transaction.
type apmSpanWrapperContext struct {
	span        *apm.Span
	transaction *apm.Transaction
}

// TraceContext returns ctx.span.TraceContext(). This is used to set the
// parent trace context for spans created through the OpenTracing API.
func (ctx apmSpanWrapperContext) TraceContext() apm.TraceContext {
	return ctx.span.TraceContext()
}

// Transaction returns ctx.transaction. This is used to obtain the transaction
// to use for creating spans through the OpenTracing API.
func (ctx apmSpanWrapperContext) Transaction() *apm.Transaction {
	return ctx.transaction
}

// ForeachBaggageItem is a no-op; we do not support baggage propagation.
func (apmSpanWrapperContext) ForeachBaggageItem(handler func(k, v string) bool) {}

// apmSpanWrapper is an opentracing.Span that wraps an apmSpanWrapperContext.
type apmSpanWrapper struct {
	apmBaseWrapper
	spanContext apmSpanWrapperContext
}

// Span returns s.spanContext.span. This is used by spanFromContext.
func (s apmSpanWrapper) Span() *apm.Span {
	return s.spanContext.span
}

// SetOperationName sets or changes the operation name.
func (s apmSpanWrapper) SetOperationName(operationName string) opentracing.Span {
	return s
}

// SetTag adds or changes a tag.
func (s apmSpanWrapper) SetTag(key string, value interface{}) opentracing.Span {
	return s
}

// Context returns the span's current context.
//
// It is valid to call Context after calling Finish or FinishWithOptions.
// The resulting context is also valid after the span is finished.
func (s apmSpanWrapper) Context() opentracing.SpanContext {
	return s.spanContext
}

// BaggageItem returns the empty string; we do not support baggage.
func (apmSpanWrapper) BaggageItem(key string) string {
	return ""
}

// SetBaggageItem is a no-op; we do not support baggage.
func (s apmSpanWrapper) SetBaggageItem(key, val string) opentracing.Span {
	// We do not support baggage.
	return s
}

// apmTransactionWrapperContext is an opentracing.SpanContext that wraps
// an apm.Transaction.
type apmTransactionWrapperContext struct {
	transaction *apm.Transaction
}

// TraceContext returns ctx.transaction.TraceContext(). This is used to set the
// parent trace context for spans created through the OpenTracing API.
func (ctx apmTransactionWrapperContext) TraceContext() apm.TraceContext {
	return ctx.transaction.TraceContext()
}

// Transaction returns ctx.transaction. This is used to obtain the transaction
// to use for creating spans through the OpenTracing API.
func (ctx apmTransactionWrapperContext) Transaction() *apm.Transaction {
	return ctx.transaction
}

// ForeachBaggageItem is a no-op; we do not support baggage propagation.
func (apmTransactionWrapperContext) ForeachBaggageItem(handler func(k, v string) bool) {}

// apmTransactionWrapper is an opentracing.Span that wraps an apmTransactionWrapperContext.
type apmTransactionWrapper struct {
	apmBaseWrapper
	spanContext apmTransactionWrapperContext
}

// SetOperationName sets or changes the operation name.
func (s apmTransactionWrapper) SetOperationName(operationName string) opentracing.Span {
	return s
}

// SetTag adds or changes a tag.
func (s apmTransactionWrapper) SetTag(key string, value interface{}) opentracing.Span {
	return s
}

// Context returns the span's current context.
//
// It is valid to call Context after calling Finish or FinishWithOptions.
// The resulting context is also valid after the span is finished.
func (s apmTransactionWrapper) Context() opentracing.SpanContext {
	return s.spanContext
}

// BaggageItem returns the empty string; we do not support baggage.
func (apmTransactionWrapper) BaggageItem(key string) string {
	return ""
}

// SetBaggageItem is a no-op; we do not support baggage.
func (s apmTransactionWrapper) SetBaggageItem(key, val string) opentracing.Span {
	// We do not support baggage.
	return s
}

type apmBaseWrapper struct{}

// Tracer returns the Tracer that created this span.
func (apmBaseWrapper) Tracer() opentracing.Tracer {
	return opentracing.NoopTracer{}
}

// Finish ends the span; this (or FinishWithOptions) must be the last method
// call on the span, except for calls to Context which may be called at any
// time.
func (apmBaseWrapper) Finish() {}

// FinishWithOptions is like Finish, but provides explicit control over the
// end timestamp and log data.
func (apmBaseWrapper) FinishWithOptions(opentracing.FinishOptions) {}

// LogKV is a no-op for APM wrapper spans.
func (apmBaseWrapper) LogKV(keyValues ...interface{}) {
	// No-op.
}

// LogFields is a no-op for APM wrapper spans.
func (apmBaseWrapper) LogFields(fields ...log.Field) {
	// No-op.
}

// LogEvent is deprecated, and is a no-op.
func (apmBaseWrapper) LogEvent(event string) {
	// Deprecated, no-op.
}

// LogEventWithPayload is deprecated, and is a no-op.
func (apmBaseWrapper) LogEventWithPayload(event string, payload interface{}) {
	// Deprecated, no-op.
}

// Log is deprecated, and is a no-op.
func (apmBaseWrapper) Log(ld opentracing.LogData) {
	// Deprecated, no-op.
}
