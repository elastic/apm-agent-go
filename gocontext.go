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

package apm // import "go.elastic.co/apm/v2"

import (
	"context"
)

// ContextWithSpan returns a copy of parent in which the given span
// is stored, associated with the key ContextSpanKey.
func ContextWithSpan(parent context.Context, s *Span) context.Context {
	return OverrideContextWithSpan(parent, s)
}

// ContextWithTransaction returns a copy of parent in which the given
// transaction is stored, associated with the key ContextTransactionKey.
func ContextWithTransaction(parent context.Context, t *Transaction) context.Context {
	return OverrideContextWithTransaction(parent, t)
}

// ContextWithBodyCapturer returns a copy of parent in which the given
// body capturer is stored, associated with the key bodyCapturerKey.
func ContextWithBodyCapturer(parent context.Context, bc *BodyCapturer) context.Context {
	return OverrideContextWithBodyCapturer(parent, bc)
}

// SpanFromContext returns the current Span in context, if any. The span must
// have been added to the context previously using ContextWithSpan, or the
// top-level StartSpan function.
func SpanFromContext(ctx context.Context) *Span {
	return OverrideSpanFromContext(ctx)
}

// TransactionFromContext returns the current Transaction in context, if any.
// The transaction must have been added to the context previously using
// ContextWithTransaction.
func TransactionFromContext(ctx context.Context) *Transaction {
	return OverrideTransactionFromContext(ctx)
}

// BodyCapturerFromContext returns the BodyCapturer in context, if any.
// The body capturer must have been added to the context previously using
// ContextWithBodyCapturer.
func BodyCapturerFromContext(ctx context.Context) *BodyCapturer {
	return OverrideBodyCapturerFromContext(ctx)
}

// DetachedContext returns a new context detached from the lifetime
// of ctx, but which still returns the values of ctx.
//
// DetachedContext can be used to maintain the trace context required
// to correlate events, but where the operation is "fire-and-forget",
// and should not be affected by the deadline or cancellation of ctx.
func DetachedContext(ctx context.Context) context.Context {
	return &detachedContext{Context: context.Background(), orig: ctx}
}

type detachedContext struct {
	context.Context
	orig context.Context
}

// Value returns c.orig.Value(key).
func (c *detachedContext) Value(key interface{}) interface{} {
	return c.orig.Value(key)
}

// StartSpan is equivalent to calling StartSpanOptions with a zero SpanOptions struct.
func StartSpan(ctx context.Context, name, spanType string) (*Span, context.Context) {
	return StartSpanOptions(ctx, name, spanType, SpanOptions{})
}

// StartSpanOptions starts and returns a new Span within the sampled transaction
// and parent span in the context, if any. If the span isn't dropped, it will be
// stored in the resulting context.
//
// If opts.Parent is non-zero, its value will be used in preference to any parent
// span in ctx.
//
// StartSpanOptions always returns a non-nil Span. Its End method must be called
// when the span completes.
func StartSpanOptions(ctx context.Context, name, spanType string, opts SpanOptions) (*Span, context.Context) {
	var span *Span
	if opts.parent = SpanFromContext(ctx); opts.parent != nil {
		if opts.parent.tx == nil && opts.parent.tracer != nil {
			span = opts.parent.tracer.StartSpan(name, spanType, opts.parent.transactionID, opts)
		} else {
			span = opts.parent.tx.StartSpanOptions(name, spanType, opts)
		}
	} else {
		tx := TransactionFromContext(ctx)
		span = tx.StartSpanOptions(name, spanType, opts)
	}
	if !span.Dropped() {
		ctx = ContextWithSpan(ctx, span)
	}
	return span, ctx
}

// CaptureError returns a new Error related to the sampled transaction
// and span present in the context, if any, and sets its exception info
// from err. The Error.Handled field will be set to true, and a stacktrace
// set either from err, or from the caller.
//
// If the provided error is nil, then CaptureError will also return nil;
// otherwise a non-nil Error will always be returned. If there is no
// transaction or span in the context, then the returned Error's Send
// method will have no effect.
func CaptureError(ctx context.Context, err error) *Error {
	if err == nil {
		return nil
	}
	if span := SpanFromContext(ctx); span != nil {
		if span.tracer == nil {
			return &Error{cause: err, err: err.Error()}
		}
		e := span.tracer.NewError(err)
		e.Handled = true
		e.SetSpan(span)
		return e
	} else if tx := TransactionFromContext(ctx); tx != nil {
		if tx.tracer == nil {
			return &Error{cause: err, err: err.Error()}
		}
		e := tx.tracer.NewError(err)
		e.Handled = true
		bc := BodyCapturerFromContext(ctx)
		if bc != nil {
			e.Context.SetHTTPRequest(bc.request)
			e.Context.SetHTTPRequestBody(bc)
		}
		e.SetTransaction(tx)
		return e
	} else {
		return &Error{cause: err, err: err.Error()}
	}
}

var (
	// OverrideContextWithSpan returns a copy of parent in which the given
	// span is stored, associated with the key ContextSpanKey.
	//
	// OverrideContextWithSpan is a variable to allow other packages, such
	// as apmot, to replace it at package init time.
	OverrideContextWithSpan = defaultContextWithSpan

	// OverrideContextWithTransaction returns a copy of parent in which the
	// given transaction is stored, associated with the key
	// ContextTransactionKey.
	//
	// ContextWithTransaction is a variable to allow other packages, such as
	// apmot, to replace it at package init time.
	OverrideContextWithTransaction = defaultContextWithTransaction

	// OverrideContextWithBodyCapturer returns a copy of parent in which the
	// given body capturer is stored, associated with the key
	// bodyCapturerKey.
	//
	// OverrideContextWithBodyCapturer is a variable to allow other packages,
	// such as apmot, to replace it at package init time.
	OverrideContextWithBodyCapturer = defaultContextWithBodyCapturer

	// OverrideSpanFromContext returns the current Span in context, if any.
	// The span must have been added to the context previously using
	// ContextWithSpan, or the top-level StartSpan function.
	//
	// SpanFromContext is a variable to allow other packages, such as apmot,
	// to replace it at package init time.
	OverrideSpanFromContext = defaultSpanFromContext

	// OverrideTransactionFromContext returns the current Transaction in
	// context, if any. The transaction must have been added to the context
	// previously using ContextWithTransaction.
	//
	// OverrideTransactionFromContext is a variable to allow other packages,
	// such as apmot, to replace it at package init time.
	OverrideTransactionFromContext = defaultTransactionFromContext

	// OverrideBodyCapturerFromContext returns the BodyCapturer in context,
	// if any. The body capturer must have been added to the context
	// previously using ContextWithBodyCapturer.
	//
	// OverrideBodyCapturerFromContext is a variable to allow other
	// packages, such as apmot, to replace it at package init time.
	OverrideBodyCapturerFromContext = defaultBodyCapturerFromContext
)

type spanKey struct{}
type transactionKey struct{}
type bodyCapturerKey struct{}

// defaultContextWithSpan is the default value for ContextWithSpan.
func defaultContextWithSpan(ctx context.Context, span *Span) context.Context {
	return context.WithValue(ctx, spanKey{}, span)
}

// defaultContextWithTransaction is the default value for ContextWithTransaction.
func defaultContextWithTransaction(ctx context.Context, tx *Transaction) context.Context {
	return context.WithValue(ctx, transactionKey{}, tx)
}

// defaultContextWithBodyCapturer is the default value for ContextWithBodyCapturer.
func defaultContextWithBodyCapturer(ctx context.Context, bc *BodyCapturer) context.Context {
	return context.WithValue(ctx, bodyCapturerKey{}, bc)
}

// defaultSpanFromContext is the default value for SpanFromContext.
func defaultSpanFromContext(ctx context.Context) *Span {
	span, _ := ctx.Value(spanKey{}).(*Span)
	return span
}

// defaultTransactionFromContext is the default value for TransactionFromContext.
func defaultTransactionFromContext(ctx context.Context) *Transaction {
	tx, _ := ctx.Value(transactionKey{}).(*Transaction)
	return tx
}

// defaultBodyCapturerFromContext is the default value for BodyCapturerFromContext.
func defaultBodyCapturerFromContext(ctx context.Context) *BodyCapturer {
	bc, _ := ctx.Value(bodyCapturerKey{}).(*BodyCapturer)
	return bc
}
