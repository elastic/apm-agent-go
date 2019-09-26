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

package apmzerolog

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"

	"go.elastic.co/apm"
	"go.elastic.co/apm/stacktrace"
)

const (
	// DefaultFatalFlushTimeout is the default value for Writer.FatalFlushTimeout.
	DefaultFatalFlushTimeout = 5 * time.Second

	// StackSourceLineName is the key for the line number of a stack frame.
	StackSourceLineName = "line"

	// StackSourceFunctionName is the key for the function name of a stack frame.
	StackSourceFunctionName = "func"
)

func init() {
	stacktrace.RegisterLibraryPackage("github.com/rs/zerolog")
}

// Writer is an implementation of zerolog.LevelWriter, reporting log records as
// errors to the APM Server. If TraceContext is used to add trace IDs to the log
// records, the errors reported will be associated with them.
//
// Because we only have access to the serialised form of the log record, we must
// rely on enough information being encoded into the events. For error stack traces,
// you must use zerolog's Stack() method, and set zerolog.ErrorStackMarshaler
// either to github.com/rs/zerolog/pkgerrors.MarshalStack, or to the function
// apmzerolog.MarshalErrorStack in this package. The pkgerrors.MarshalStack
// implementation omits some information, whereas apmzerolog is designed to
// convey the complete file location and fully qualified function name.
type Writer struct {
	// Tracer is the apm.Tracer to use for reporting errors.
	// If Tracer is nil, then apm.DefaultTracer will be used.
	Tracer *apm.Tracer

	// FatalFlushTimeout is the amount of time to wait while
	// flushing a fatal log message to the APM Server before
	// the process is exited. If this is 0, then
	// DefaultFatalFlushTimeout will be used. If the timeout
	// is a negative value, then no flushing will be performed.
	FatalFlushTimeout time.Duration

	// MinLevel holds the minimum level of logs to send to
	// Elastic APM as errors.
	//
	// MinLevel must be greater than or equal to zerolog.ErrorLevel.
	// If it is less than this, zerolog.ErrorLevel will be used as
	// the minimum instead.
	MinLevel zerolog.Level
}

func (w *Writer) tracer() *apm.Tracer {
	tracer := w.Tracer
	if tracer == nil {
		tracer = apm.DefaultTracer
	}
	return tracer
}

func (w *Writer) minLevel() zerolog.Level {
	minLevel := w.MinLevel
	if minLevel < zerolog.ErrorLevel {
		minLevel = zerolog.ErrorLevel
	}
	return minLevel
}

// Write is a no-op.
func (*Writer) Write(p []byte) (int, error) {
	return len(p), nil
}

// WriteLevel decodes the JSON-encoded log record in p, and reports it as an error using w.Tracer.
func (w *Writer) WriteLevel(level zerolog.Level, p []byte) (int, error) {
	if level < w.minLevel() || level >= zerolog.NoLevel {
		return len(p), nil
	}
	tracer := w.tracer()
	if !tracer.Active() {
		return len(p), nil
	}
	var logRecord logRecord
	if err := logRecord.decode(bytes.NewReader(p)); err != nil {
		return 0, err
	}

	errlog := tracer.NewErrorLog(apm.ErrorLogRecord{
		Level:   level.String(),
		Message: logRecord.message,
		Error:   logRecord.err,
	})
	if !logRecord.timestamp.IsZero() {
		errlog.Timestamp = logRecord.timestamp
	}
	errlog.Handled = true
	errlog.SetStacktrace(1)
	errlog.TraceID = logRecord.traceID
	errlog.TransactionID = logRecord.transactionID
	if logRecord.spanID.Validate() == nil {
		errlog.ParentID = logRecord.spanID
	} else {
		errlog.ParentID = logRecord.transactionID
	}
	errlog.Send()

	if level == zerolog.FatalLevel {
		// Zap will exit the process following a fatal log message, so we flush the tracer.
		flushTimeout := w.FatalFlushTimeout
		if flushTimeout == 0 {
			flushTimeout = DefaultFatalFlushTimeout
		}
		if flushTimeout >= 0 {
			ctx, cancel := context.WithTimeout(context.Background(), flushTimeout)
			defer cancel()
			tracer.Flush(ctx.Done())
		}
	}
	return len(p), nil
}

type logRecord struct {
	message               string
	timestamp             time.Time
	err                   error
	traceID               apm.TraceID
	transactionID, spanID apm.SpanID
}

func (l *logRecord) decode(r io.Reader) (result error) {
	m := make(map[string]interface{})
	d := json.NewDecoder(r)
	d.UseNumber()
	if err := d.Decode(&m); err != nil {
		return err
	}

	l.message, _ = m[zerolog.MessageFieldName].(string)
	if strval, ok := m[zerolog.TimestampFieldName].(string); ok {
		if t, err := time.Parse(zerolog.TimeFieldFormat, strval); err == nil {
			l.timestamp = t.UTC()
		}
	}
	if errmsg, ok := m[zerolog.ErrorFieldName].(string); ok {
		err := &jsonError{message: errmsg}
		if stack, ok := m[zerolog.ErrorStackFieldName].([]interface{}); ok {
			frames := make([]stacktrace.Frame, 0, len(stack))
			for i := range stack {
				in, ok := stack[i].(map[string]interface{})
				if !ok {
					continue
				}
				var frame stacktrace.Frame
				frame.File, _ = in[pkgerrors.StackSourceFileName].(string)
				frame.Function, _ = in[StackSourceFunctionName].(string)
				if strval, ok := in[StackSourceLineName].(string); ok {
					if line, err := strconv.Atoi(strval); err == nil {
						frame.Line = line
					}
				}
				frames = append(frames, frame)
			}
			err.stack = frames
		}
		l.err = err
	}

	if strval, ok := m[SpanIDFieldName].(string); ok {
		if err := decodeHex(l.spanID[:], strval); err != nil {
			return errors.Wrap(err, "invalid span.id")
		}
	}

	if strval, ok := m[TraceIDFieldName].(string); ok {
		if err := decodeHex(l.traceID[:], strval); err != nil {
			return errors.Wrap(err, "invalid trace.id")
		}
	}
	if strval, ok := m[TransactionIDFieldName].(string); ok {
		if err := decodeHex(l.transactionID[:], strval); err != nil {
			return errors.Wrap(err, "invalid transaction.id")
		}
	}
	return nil
}

func decodeHex(out []byte, in string) error {
	if n := hex.EncodedLen(len(out)); n != len(in) {
		return errors.Errorf(
			"invalid value length (expected %d bytes, got %d)",
			n, len(in),
		)
	}
	_, err := hex.Decode(out, []byte(in))
	return err
}

type jsonError struct {
	message string
	stack   []stacktrace.Frame
}

func (e *jsonError) Type() string {
	return "error"
}

func (e *jsonError) Error() string {
	return e.message
}

func (e *jsonError) StackTrace() []stacktrace.Frame {
	return e.stack
}
