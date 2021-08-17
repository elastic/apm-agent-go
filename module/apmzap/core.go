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

//go:build go1.9
// +build go1.9

package apmzap // import "go.elastic.co/apm/module/apmzap"

import (
	"context"
	"time"

	"go.uber.org/zap/zapcore"

	"go.elastic.co/apm"
	"go.elastic.co/apm/stacktrace"
)

const (
	// DefaultFatalFlushTimeout is the default value for Hook.FatalFlushTimeout.
	DefaultFatalFlushTimeout = 5 * time.Second
)

func init() {
	stacktrace.RegisterLibraryPackage("go.uber.org/zap")
}

// Core is an implementation of zapcore.Core, reporting log records as
// errors to the APM Server. If TraceContext is used to add trace IDs
// to the log records, the errors reported will be associated with them.
type Core struct {
	// Tracer is the apm.Tracer to use for reporting errors.
	// If Tracer is nil, then apm.DefaultTracer will be used.
	Tracer *apm.Tracer

	// FatalFlushTimeout is the amount of time to wait while
	// flushing a fatal log message to the APM Server before
	// the process is exited. If this is 0, then
	// DefaultFatalFlushTimeout will be used. If the timeout
	// is a negative value, then no flushing will be performed.
	FatalFlushTimeout time.Duration
}

func (c *Core) tracer() *apm.Tracer {
	tracer := c.Tracer
	if tracer == nil {
		tracer = apm.DefaultTracer
	}
	return tracer
}

// WrapCore returns zapcore.NewTee(core, c).
// WrapCore is suitable for passing to zap.WrapCore.
func (c *Core) WrapCore(core zapcore.Core) zapcore.Core {
	return zapcore.NewTee(core, c)
}

// Sync is a no-op.
func (*Core) Sync() error {
	return nil
}

// Enabled returns true if level is >= zapcore.ErrorLevel.
func (*Core) Enabled(level zapcore.Level) bool {
	return level >= zapcore.ErrorLevel
}

// With returns a new zapcore.Core that decorates c with fields.
func (c *Core) With(fields []zapcore.Field) zapcore.Core {
	out := &contextCore{core: c}
	out.traceContext.fields(fields)
	return out
}

// Check checks if the entry should be logged, and adds c to checked if so.
func (c *Core) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if entry.Level < zapcore.ErrorLevel || !c.tracer().Recording() {
		return checked
	}
	return checked.AddCore(entry, c)
}

// Write reports entry and fields as an error using c.tracer.
func (c *Core) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	core := contextCore{core: c}
	return core.Write(entry, fields)
}

type contextCore struct {
	core         *Core
	traceContext traceContext
}

func (c *contextCore) Sync() error {
	return nil
}

func (c *contextCore) Enabled(level zapcore.Level) bool {
	return level >= zapcore.ErrorLevel
}

func (c *contextCore) With(fields []zapcore.Field) zapcore.Core {
	newCore := &contextCore{
		core:         c.core,
		traceContext: c.traceContext,
	}
	newCore.traceContext.fields(fields)
	return newCore
}

func (c *contextCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if entry.Level < zapcore.ErrorLevel || !c.core.tracer().Recording() {
		return checked
	}
	return checked.AddCore(entry, c)
}

func (c *contextCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	traceContext := c.traceContext
	traceContext.fields(fields)

	tracer := c.core.tracer()
	errlog := tracer.NewErrorLog(apm.ErrorLogRecord{
		Message:    entry.Message,
		Level:      entry.Level.String(),
		LoggerName: entry.LoggerName,
		Error:      traceContext.err,
	})
	errlog.Handled = true
	errlog.Timestamp = entry.Time
	errlog.SetStacktrace(1)
	errlog.TraceID = traceContext.traceID
	errlog.TransactionID = traceContext.transactionID
	if traceContext.spanID.Validate() == nil {
		errlog.ParentID = traceContext.spanID
	} else {
		errlog.ParentID = traceContext.transactionID
	}
	errlog.Send()

	if entry.Level == zapcore.FatalLevel {
		// Zap will exit the process following a fatal log message, so we flush the tracer.
		flushTimeout := c.core.FatalFlushTimeout
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

type traceContext struct {
	err                   error
	traceID               apm.TraceID
	transactionID, spanID apm.SpanID
}

func (c *traceContext) fields(fields []zapcore.Field) {
	for _, field := range fields {
		switch field.Key {
		case "error":
			c.err, _ = field.Interface.(error)
		case FieldKeyTraceID:
			c.traceID, _ = field.Interface.(apm.TraceID)
		case FieldKeyTransactionID:
			c.transactionID, _ = field.Interface.(apm.SpanID)
		case FieldKeySpanID:
			c.spanID, _ = field.Interface.(apm.SpanID)
		}
	}
}
