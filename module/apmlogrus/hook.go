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

package apmlogrus // import "go.elastic.co/apm/module/apmlogrus"

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"go.elastic.co/apm"
	"go.elastic.co/apm/stacktrace"
)

var (
	// DefaultLogLevels is the log levels for which errors are reported by Hook, if Hook.LogLevels is not set.
	DefaultLogLevels = []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
	}
)

const (
	// DefaultFatalFlushTimeout is the default value for Hook.FatalFlushTimeout.
	DefaultFatalFlushTimeout = 5 * time.Second
)

func init() {
	stacktrace.RegisterLibraryPackage("github.com/sirupsen/logrus")
}

// Hook implements logrus.Hook, reporting log records as errors
// to the APM Server. If TraceContext is used to add trace IDs
// to the log records, the errors reported will be associated
// with them.
type Hook struct {
	// Tracer is the apm.Tracer to use for reporting errors.
	// If Tracer is nil, then apm.DefaultTracer will be used.
	Tracer *apm.Tracer

	// LogLevels holds the log levels to report as errors.
	// If LogLevels is nil, then the DefaultLogLevels will
	// be used.
	LogLevels []logrus.Level

	// FatalFlushTimeout is the amount of time to wait while
	// flushing a fatal log message to the APM Server before
	// the process is exited. If this is 0, then
	// DefaultFatalFlushTimeout will be used. If the timeout
	// is a negative value, then no flushing will be performed.
	FatalFlushTimeout time.Duration
}

func (h *Hook) tracer() *apm.Tracer {
	tracer := h.Tracer
	if tracer == nil {
		tracer = apm.DefaultTracer
	}
	return tracer
}

// Levels returns h.LogLevels, satisfying the logrus.Hook interface.
func (h *Hook) Levels() []logrus.Level {
	levels := h.LogLevels
	if levels == nil {
		levels = DefaultLogLevels
	}
	return levels
}

// Fire reports the log entry as an error to the APM Server.
func (h *Hook) Fire(entry *logrus.Entry) error {
	tracer := h.tracer()
	if !tracer.Recording() {
		return nil
	}

	err, _ := entry.Data[logrus.ErrorKey].(error)
	errlog := tracer.NewErrorLog(apm.ErrorLogRecord{
		Message: entry.Message,
		Level:   entry.Level.String(),
		Error:   err,
	})
	errlog.Handled = true
	errlog.Timestamp = entry.Time
	errlog.SetStacktrace(1)

	// Extract trace context added with apmlogrus.TraceContext,
	// and include it in the reported error.
	if traceID, ok := entry.Data[FieldKeyTraceID].(apm.TraceID); ok {
		errlog.TraceID = traceID
	}
	if transactionID, ok := entry.Data[FieldKeyTransactionID].(apm.SpanID); ok {
		errlog.TransactionID = transactionID
		errlog.ParentID = transactionID
	}
	if spanID, ok := entry.Data[FieldKeySpanID].(apm.SpanID); ok {
		errlog.ParentID = spanID
	}

	errlog.Send()
	if entry.Level == logrus.FatalLevel {
		// In its default configuration, logrus will exit the process
		// following a fatal log message, so we flush the tracer.
		flushTimeout := h.FatalFlushTimeout
		if flushTimeout == 0 {
			flushTimeout = DefaultFatalFlushTimeout
		}
		if flushTimeout >= 0 {
			ctx, cancel := context.WithTimeout(context.Background(), flushTimeout)
			defer cancel()
			tracer.Flush(ctx.Done())
		}
	}
	return nil
}
