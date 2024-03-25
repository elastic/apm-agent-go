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
)

type ApmHandler struct {
	Tracer       *apm.Tracer
	ReportLevels []slog.Level
	Handler      slog.Handler
}

func (s *ApmHandler) tracer() *apm.Tracer {
	if s.Tracer == nil {
		return apm.DefaultTracer()
	}
	return s.Tracer
}

func (s *ApmHandler) levels() []slog.Level {
	if s.ReportLevels == nil {
		return []slog.Level{slog.LevelError}
	}
	return s.ReportLevels
}

// Enabled reports whether the handler handles records at the given level.
func (s *ApmHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return s.Handler.Enabled(ctx, level)
}

// WithAttrs returns a new ApmHandler with passed attributes attached.
func (s *ApmHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ApmHandler{s.Tracer, s.ReportLevels, s.Handler.WithAttrs(attrs)}
}

// WithGroup returns a new ApmHandler with passed group attached.
func (s *ApmHandler) WithGroup(name string) slog.Handler {
	return &ApmHandler{s.Tracer, s.ReportLevels, s.Handler.WithGroup(name)}
}

func (s *ApmHandler) Handle(ctx context.Context, r slog.Record) error {

	// report record as APM error
	tracer := s.tracer()
	if slices.Contains(s.levels(), r.Level) && tracer.Recording() {

		// attempt to find error/err attribute
		// slog doesnt have a standard way of attaching an
		// error to a record, so attempting to grab any attribute
		// that has error/err as key and extracting the value
		// seems like a likely way to do it.
		var err error
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == "error" || a.Key == "err" {
				if v, ok := a.Value.Any().(error); ok {
					err = v
					return false
				}
				if v, ok := a.Value.Any().(string); ok {
					err = errors.New(v)
					return false
				}
				return false
			}
			return true
		})
		// if error/err attribute exists, use it as Error value
		var errLogRecord apm.ErrorLogRecord
		if err != nil {
			errLogRecord = apm.ErrorLogRecord{
				Message: r.Message,
				Level:   strings.ToLower(r.Level.String()),
				Error:   err,
			}
		} else {
			errLogRecord = apm.ErrorLogRecord{
				Message: r.Message,
				Level:   strings.ToLower(r.Level.String()),
			}
		}

		errlog := tracer.NewErrorLog(errLogRecord)
		errlog.Handled = true
		errlog.Timestamp = r.Time.UTC()
		errlog.SetStacktrace(2)

        // extract available trace info from context
		if tx := apm.TransactionFromContext(ctx); tx != nil {
			errlog.TraceID = tx.TraceContext().Trace
			errlog.TransactionID = tx.TraceContext().Span
			errlog.ParentID = tx.TraceContext().Span
		}
		if span := apm.SpanFromContext(ctx); span != nil {
			errlog.ParentID = span.TraceContext().Span
		}
		errlog.Send()
	}

	// attach trace context if exists and attach to record
	if tx := apm.TransactionFromContext(ctx); tx != nil {
		r.Add(FieldKeyTraceID, tx.TraceContext().Trace)
		r.Add(FieldKeyTransactionID, tx.TraceContext().Span)
	}
	if span := apm.SpanFromContext(ctx); span != nil {
		r.Add(FieldKeySpanID, span.TraceContext().Span)
	}

	return s.Handler.Handle(ctx, r)
}

type apmHandlerOption func(h *ApmHandler)

func NewApmHandler(opts ...apmHandlerOption) *ApmHandler {
	h := &ApmHandler{apm.DefaultTracer(), []slog.Level{slog.LevelError}, slog.Default().Handler()}
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
