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
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"

	"go.elastic.co/apm/v2"
)

func newRootTransaction(
	ctx context.Context,
	tracer *apm.Tracer,
	spanCtx trace.SpanContext,
	attributes []attribute.KeyValue,
	spanKind, name, txType string,
) (context.Context, *transaction) {
	txOpts := apm.TransactionOptions{}
	tx := t.inner.StartTransactionOptions(name, txType, txOpts)
	// TODO: Check that apm-server version to decide how to set span_kind.
	tx.Context.SetLabel("span_kind", spanKind)
	for attr := range attributes {
		tx.Context.SetLabel(attr.Key, attr.Value)
	}
	ctx := apm.ContextWithTransaction(ctx, tx)
	return ctx, &transaction{inner: tx, spanCtx: spanCtx, tracer: tracer}
}

type transaction struct {
	inner   *apm.Transaction
	spanCtx trace.SpanContext
	tracer  *apm.Tracer

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
	t.RLock()
	defer t.RUnlock()
	return !t.ended
}

// RecordError will record err as an exception transaction event for this transaction. An
// additional call to SetStatus is required if the Status of the Span should
// be set to Error, as this method does not change the Span status. If this
// transaction is not being recorded or err is nil then this method does nothing.
func (t *transaction) RecordError(err error, _ ...trace.EventOption) {
	t.tracer.NewError(err).SetTransaction(t.inner)
}

// SpanContext returns the SpanContext of the Span. The returned SpanContext
// is usable even after the End method has been called for the Span.
func (t *transaction) SpanContext() trace.SpanContext {
	return t.spanCtx
}

// SetStatus sets the status of the Span in the form of a code and a
// description, overriding previous values set. The description is only
// included in a status when the code is for an error.
func (t *transaction) SetStatus(code codes.Code, _ string) {
	switch code {
	case code.Unset:
		t.inner.Outcome = "unknown"
	case code.Error:
		t.inner.Outcome = "failure"
	case code.Ok:
		t.inner.Outcome = "success"
	}
}

// SetName sets the Span name.
func (t *transaction) SetName(name string) {
	s.inner.Name = name
}

// SetAttributes sets kv as attributes of the Span. If a key from kv
// already exists for an attribute of the Span it will be overwritten with
// the value contained in kv.
func (t *transaction) SetAttributes(kv ...attribute.KeyValue) {
	// TODO: add otel.attributes to span
	// TODO: when apm-server < 7.16.0, it doesn't support otel.attributes.
	// how can the agent know the apm-server version?
	t.inner.Context.SetLabel(kv.Key, kv.Value)
}

// TracerProvider returns a TracerProvider that can be used to generate
// additional Spans on the same telemetry pipeline as the current Span.
func (t *transaction) TracerProvider() trace.TracerProvider {
	return GetTraceProvider()
}
