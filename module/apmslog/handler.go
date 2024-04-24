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
	tracer           *apm.Tracer
	reportLevels     []slog.Level
	errorRecordAttrs []string
	handler          slog.Handler
}

// Enabled reports whether the handler handles records at the given level.
func (h *ApmHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// WithAttrs returns a new ApmHandler with passed attributes attached.
func (h *ApmHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ApmHandler{h.tracer, h.reportLevels, h.errorRecordAttrs, h.handler.WithAttrs(attrs)}
}

// WithGroup returns a new ApmHandler with passed group attached.
func (h *ApmHandler) WithGroup(name string) slog.Handler {
	return &ApmHandler{h.tracer, h.reportLevels, h.errorRecordAttrs, h.handler.WithGroup(name)}
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
	if h.tracer != nil && h.tracer.Recording() && slices.Contains(h.reportLevels, r.Level) {

		// attempt to find error attributes
		// slog doesnt have a standard way of attaching an
		// error to a record, so attempting to grab any attribute
		// that has error/err keys OR keys user has defined as reportable
		// and extracting the values seems like a likely way to do it.
		errorsToAttach := []error{}
		r.Attrs(func(a slog.Attr) bool {
			if slices.Contains(h.errorRecordAttrs, a.Key) {
				var err error
				// first check if value is of error type to retain as much info as possible
				if v, ok := a.Value.Any().(error); ok {
					errorsToAttach = append(errorsToAttach, v)
					// else just convert reportable error value as string
				} else {
					errorsToAttach = append(errorsToAttach, errors.Join(err, fmt.Errorf("%s", a.Value.String())))
				}
			}
			return true
		})

		// If there are multiple reportable error attributes, create a new
		// apm.ErrorLogRecord for each. Otherwise just create one apm.ErrorLogRecord
		// with no Error.
		errLogRecords := []apm.ErrorLogRecord{}
		if len(errorsToAttach) == 0 {
			errRecord := apm.ErrorLogRecord{
				Message: r.Message,
				Level:   strings.ToLower(r.Level.String()),
			}
			errLogRecords = append(errLogRecords, errRecord)
		} else {
			for _, err := range errorsToAttach {
				errRecord := apm.ErrorLogRecord{
					Message: r.Message,
					Level:   strings.ToLower(r.Level.String()),
					Error:   err,
				}
				errLogRecords = append(errLogRecords, errRecord)
			}
		}

		// for each errRecord, send to apm
		for _, errRecord := range errLogRecords {
			errlog := h.tracer.NewErrorLog(errRecord)
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
	}

	return h.handler.Handle(ctx, r)
}

type apmHandlerOption func(h *ApmHandler)

// Create a new ApmHandler.
func NewApmHandler(opts ...apmHandlerOption) *ApmHandler {
	h := &ApmHandler{
		apm.DefaultTracer(),
		[]slog.Level{slog.LevelError},
		[]string{SlogErrorKeyErr, SlogErrorKeyError},
		slog.Default().Handler(),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Set slog handler for ApmHandler
// default: slog.Default().Handler()
func WithHandler(handler slog.Handler) apmHandlerOption {
	return func(h *ApmHandler) {
		h.handler = handler
	}
}

// Set which slog log level will be reported
// default: slog.LevelError
func WithReportLevel(lvls []slog.Level) apmHandlerOption {
	return func(h *ApmHandler) {
		h.reportLevels = lvls
	}
}

// Set with slog attribute keys will be used as errors.
// default: 'error','err'
func WithErrorRecordAttrs(keys []string) apmHandlerOption {
	return func(h *ApmHandler) {
		h.errorRecordAttrs = keys
	}
}

// Set custom tracer for ApmHandler.
// default: apm.DefaultTracer()
func WithTracer(tracer *apm.Tracer) apmHandlerOption {
	return func(h *ApmHandler) {
		h.tracer = tracer
	}
}
