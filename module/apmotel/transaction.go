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
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"go.elastic.co/apm/v2"
)

func newRootTransaction(
	ctx context.Context,
	tracer *apm.Tracer,
	attributes []attribute.KeyValue,
	spanKind, name, txType string,
) (context.Context, *transaction) {
	txOpts := apm.TransactionOptions{}
	tx := tracer.StartTransactionOptions(name, txType, txOpts)
	tx.Context.SetOTelSpanKind(spanKind)
	ctx = apm.ContextWithTransaction(ctx, tx)
	otelTx := &transaction{inner: tx, tracer: tracer}
	otelTx.SetAttributes(attributes...)
	return ctx, &transaction{inner: tx, tracer: tracer}
}

type transaction struct {
	inner  *apm.Transaction
	tracer *apm.Tracer

	mu    sync.RWMutex
	ended bool
}

// End completes the Span. The Span is considered complete and ready to be
// delivered through the rest of the telemetry pipeline after this method
// is called. Therefore, updates to the Span are not allowed after this
// method has been called.
func (t *transaction) End(_ ...trace.SpanEndOption) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.inner.End()
	t.ended = true
}

// AddEvent adds an event with the provided name and options.
func (t *transaction) AddEvent(name string, options ...trace.EventOption) {}

// IsRecording returns the recording state of the Span. It will return
// true if the Span is active and events can be recorded.
func (t *transaction) IsRecording() bool {
	if t == nil {
		return false
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return !t.ended
}

// RecordError will record err as an exception transaction event for this transaction. An
// additional call to SetStatus is required if the Status of the Span should
// be set to Error, as this method does not change the Span status. If this
// transaction is not being recorded or err is nil then this method does nothing.
func (t *transaction) RecordError(err error, _ ...trace.EventOption) {
	e := t.tracer.NewError(err)
	e.SetTransaction(t.inner)
	e.Send()
}

// SpanContext returns the SpanContext of the Span. The returned SpanContext
// is usable even after the End method has been called for the Span.
func (t *transaction) SpanContext() trace.SpanContext {
	traceCtx := t.inner.TraceContext()
	spanCtx := trace.SpanContext{}
	spanCtx.WithTraceID(trace.TraceID(traceCtx.Trace))
	spanCtx.WithSpanID(trace.SpanID(traceCtx.Span))
	spanCtx.WithTraceFlags(trace.TraceFlags(traceCtx.Options))
	if ts, err := trace.ParseTraceState(traceCtx.State.String()); err == nil {
		spanCtx.WithTraceState(ts)
	}
	return spanCtx

}

// SetStatus sets the status of the Span in the form of a code and a
// description, overriding previous values set. The description is only
// included in a status when the code is for an error.
func (t *transaction) SetStatus(code codes.Code, _ string) {
	switch code {
	case codes.Unset:
		t.inner.Outcome = "unknown"
	case codes.Error:
		t.inner.Outcome = "failure"
	case codes.Ok:
		t.inner.Outcome = "success"
	}
}

// SetName sets the Span name.
func (t *transaction) SetName(name string) {
	t.inner.Name = name
}

// SetAttributes sets kv as attributes of the Span. If a key from kv
// already exists for an attribute of the Span it will be overwritten with
// the value contained in kv.
func (t *transaction) SetAttributes(kvs ...attribute.KeyValue) {
	m := make(map[string]interface{}, len(kvs))
	for _, kv := range kvs {
		m[string(kv.Key)] = kv.Value.AsInterface()
	}
	t.inner.Context.SetOTelAttributes(m)

}

// TracerProvider returns a TracerProvider that can be used to generate
// additional Spans on the same telemetry pipeline as the current Span.
func (t *transaction) TracerProvider() trace.TracerProvider {
	return GetTracerProvider()
}

// Transaction returns s.inner, the underlying apm.Span. This is used to
// satisfy TransactionFromContext.
func (t *transaction) Transaction() *apm.Transaction {
	return t.inner
}
