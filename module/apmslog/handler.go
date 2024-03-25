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

package apmslog // import "go.elastic.co/apm/module/apmslog/v2"

import (
	"context"
	"errors"
	"fmt"

	"log/slog"
	"slices"
	"strings"

	"go.elastic.co/apm/v2"
)

const (
	// FieldKeyTraceID is the field key for the trace ID.
	FieldKeyTraceID = "trace.id"

	// FieldKeyTransactionID is the field key for the transaction ID.
	FieldKeyTransactionID = "transaction.id"

	// FieldKeySpanID is the field key for the span ID.
	FieldKeySpanID = "span.id"

	// SlogErrorKey* are the key name values that are reported as APM Errors
	SlogErrorKeyErr   = "err"
	SlogErrorKeyError = "error"
)

type ApmHandler struct {
	Tracer       *apm.Tracer
	ReportLevels []slog.Level
	Handler      slog.Handler
}

func (h *ApmHandler) tracer() *apm.Tracer {
	if h.Tracer == nil {
		return apm.DefaultTracer()
	}
	return h.Tracer
}

// Enabled reports whether the handler handles records at the given level.
func (h *ApmHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.Handler.Enabled(ctx, level)
}

// WithAttrs returns a new ApmHandler with passed attributes attached.
func (h *ApmHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ApmHandler{h.Tracer, h.ReportLevels, h.Handler.WithAttrs(attrs)}
}

// WithGroup returns a new ApmHandler with passed group attached.
func (h *ApmHandler) WithGroup(name string) slog.Handler {
	return &ApmHandler{h.Tracer, h.ReportLevels, h.Handler.WithGroup(name)}
}

func (h *ApmHandler) Handle(ctx context.Context, r slog.Record) error {

	// attempt to extract any available trace info from context
	var traceId apm.TraceID
	var transactionId apm.SpanID
	var parentId apm.SpanID
	if tx := apm.TransactionFromContext(ctx); tx != nil {
		traceId = tx.TraceContext().Trace
		transactionId = tx.TraceContext().Span
		parentId = tx.TraceContext().Span
		// add trace/transaction ids to slog record to be logged
		r.Add(FieldKeyTraceID, traceId)
		r.Add(FieldKeyTransactionID, transactionId)
	}
	if span := apm.SpanFromContext(ctx); span != nil {
		parentId = span.TraceContext().Span
		// add span id to slog record to be logged
		r.Add(FieldKeySpanID, parentId)
	}

	// report record as APM error
	tracer := h.tracer()
	if slices.Contains(h.ReportLevels, r.Level) && tracer.Recording() {

		// attempt to find error/err attribute
		// slog doesnt have a standard way of attaching an
		// error to a record, so attempting to grab any attribute
		// that has error/err as key and extracting the value
		// seems like a likely way to do it.
		var err error
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == SlogErrorKeyErr || a.Key == SlogErrorKeyError {
				if v, ok := a.Value.Any().(error); ok {
					err = v
					return false
				} else {
					err = errors.Join(err, fmt.Errorf("%s", a.Value.String()))
					return false
				}
			}
			return true
		})

		errlog := tracer.NewErrorLog(apm.ErrorLogRecord{
			Message: r.Message,
			Level:   strings.ToLower(r.Level.String()),
			Error:   err,
		})
		errlog.Handled = true
		errlog.Timestamp = r.Time.UTC()
		errlog.SetStacktrace(2)

		// add available trace info if not zero type
		if traceId != (apm.TraceID{}) {
			errlog.TraceID = traceId
		}
		if transactionId != (apm.SpanID{}) {
			errlog.TransactionID = transactionId
		}
		if parentId != (apm.SpanID{}) {
			errlog.ParentID = parentId
		}
		// send error to APM
		errlog.Send()
	}

	return h.Handler.Handle(ctx, r)
}

type apmHandlerOption func(h *ApmHandler)

func NewApmHandler(opts ...apmHandlerOption) *ApmHandler {
	h := &ApmHandler{
		apm.DefaultTracer(),
		[]slog.Level{slog.LevelError},
		slog.Default().Handler(),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func WithHandler(handler slog.Handler) apmHandlerOption {
	return func(h *ApmHandler) {
		h.Handler = handler
	}
}

func WithReportLevel(lvls []slog.Level) apmHandlerOption {
	return func(h *ApmHandler) {
		h.ReportLevels = lvls
	}
}

func WithTracer(tracer *apm.Tracer) apmHandlerOption {
	return func(h *ApmHandler) {
		h.Tracer = tracer
	}
}
