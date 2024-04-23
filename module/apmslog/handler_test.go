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

package apmslog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/module/apmslog/v2"
	"go.elastic.co/apm/v2"
	"go.elastic.co/apm/v2/transport/transporttest"
)

// it should add no trace attributes and not report an error
func TestHandlerNoContext(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()
	var buf bytes.Buffer
	h := newApmslogHandler(&buf, tracer)
	logger := slog.New(h)

	logger.Info("hello world")

	// assert msg looks as expected
	assert.Equal(t, `{"time":"1970-01-01T00:00:00Z","level":"INFO","msg":"hello world"}`+"\n", buf.String())

	tracer.Flush(nil)
	payloads := transport.Payloads()
	// assert we reported no errors to apm
	assert.Len(t, payloads.Errors, 0)
}

// Helper struct to check msg traces
type MsgWithTrace struct {
	TransactionId string `json:"transaction.id"`
	TraceId       string `json:"trace.id"`
	SpanId        string `json:"span.id"`
}

// it should add trace attributes and not report an error
func TestHandlerWithContext(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()
	var buf bytes.Buffer
	h := newApmslogHandler(&buf, tracer)
	logger := slog.New(h)

	tx := tracer.StartTransaction("name", "type")
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	span, ctx := apm.StartSpan(ctx, "name", "type")

	logger.InfoContext(ctx, "hello world")

	span.End()
	tx.End()

	var msg MsgWithTrace
	err := json.Unmarshal([]byte(buf.String()), &msg)
	assert.NoError(t, err)

	// assert that we added the correct traces to buff
	assert.Equal(t, tx.TraceContext().Trace.String(), msg.TraceId)
	assert.Equal(t, span.TraceContext().Span.String(), msg.SpanId)
	assert.Equal(t, tx.TraceContext().Span.String(), msg.TransactionId)

	// assert msg is as expected
	assert.Equal(t,
		fmt.Sprintf(`{"time":"1970-01-01T00:00:00Z","level":"INFO","msg":"hello world","trace.id":"%s","transaction.id":"%s","span.id":"%s"}`,
			msg.TraceId,
			msg.TransactionId,
			msg.SpanId,
		)+"\n",
		buf.String(),
	)

	// assert we reported no errors to apm
	tracer.Flush(nil)
	payloads := transport.Payloads()
	assert.Len(t, payloads.Errors, 0)
}

// it should add no trace attributes but still report an error
func TestHandlerErrorNoContext(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()
	var buf bytes.Buffer
	h := newApmslogHandler(&buf, tracer)
	logger := slog.New(h)

	logger.Error("hello world", "error", errors.New("new error"))

	// assert msg looks as expected
	assert.Equal(t, `{"time":"1970-01-01T00:00:00Z","level":"ERROR","msg":"hello world","error":"new error"}`+"\n", buf.String())

	// assert we reported one error to apm
	tracer.Flush(nil)
	payloads := transport.Payloads()
	assert.Len(t, payloads.Errors, 1)
	assert.Empty(t, payloads.Transactions, 0)
	assert.Empty(t, payloads.Spans, 0)

	// assert apm error fields are correct
	err0 := payloads.Errors[0]
	assert.Equal(t, "hello world", err0.Log.Message)
	assert.Equal(t, "error", err0.Log.Level)
	assert.Equal(t, "", err0.Log.LoggerName)
	assert.Equal(t, "", err0.Log.ParamMessage)
	assert.Equal(t, "TestHandlerErrorNoContext", err0.Culprit)
	assert.NotEmpty(t, err0.Log.Stacktrace)
	assert.Zero(t, err0.ParentID)
	assert.Zero(t, err0.TraceID)
	assert.Zero(t, err0.TransactionID)
}

// mock function to test apm error fields
func mockFuncError() error {
	return errors.New("new error")
}

// it should add trace attributes and report an error
func TestHandlerErrorWithContext(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()
	var buf bytes.Buffer
	h := newApmslogHandler(&buf, tracer)
	logger := slog.New(h)

	// create transactions and spans
	tx := tracer.StartTransaction("name", "type")
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	span, ctx := apm.StartSpan(ctx, "name", "type")

	logger.ErrorContext(ctx, "hello world", "error", mockFuncError())

	span.End()
	tx.End()

	var msg MsgWithTrace
	err := json.Unmarshal([]byte(buf.String()), &msg)
	assert.NoError(t, err)

	// assert that we added the correct traces to buff
	assert.Equal(t, tx.TraceContext().Trace.String(), msg.TraceId)
	assert.Equal(t, span.TraceContext().Span.String(), msg.SpanId)
	assert.Equal(t, tx.TraceContext().Span.String(), msg.TransactionId)

	// assert msg is as expected
	assert.Equal(t,
		fmt.Sprintf(`{"time":"1970-01-01T00:00:00Z","level":"ERROR","msg":"hello world","error":"new error","trace.id":"%s","transaction.id":"%s","span.id":"%s"}`,
			msg.TraceId,
			msg.TransactionId,
			msg.SpanId,
		)+"\n",
		buf.String(),
	)

	// assert we reported one error with traces to apm
	tracer.Flush(nil)
	payloads := transport.Payloads()
	assert.Len(t, payloads.Transactions, 1)
	assert.Len(t, payloads.Spans, 1)
	assert.Len(t, payloads.Errors, 1)

	// assert apm error fields are correct
	err0 := payloads.Errors[0]
	assert.Equal(t, payloads.Spans[0].ID, err0.ParentID)
	assert.Equal(t, payloads.Transactions[0].TraceID, err0.TraceID)
	assert.Equal(t, payloads.Transactions[0].ID, err0.TransactionID)
	assert.Equal(t, "new error", err0.Exception.Message)
	assert.Equal(t, "hello world", err0.Log.Message)
	assert.Equal(t, "mockFuncError", err0.Culprit)
	assert.Equal(t, "mockFuncError", err0.Exception.Stacktrace[0].Function)
	assert.Equal(t, "(*Logger).log", err0.Log.Stacktrace[0].Function)
	assert.Equal(t, "error", err0.Log.Level)
	assert.Equal(t, "", err0.Log.LoggerName)
	assert.Equal(t, "", err0.Log.ParamMessage)
	assert.NotEmpty(t, err0.Log.Stacktrace)
	assert.NotEmpty(t, err0.Exception.Stacktrace)
	assert.NotEqual(t, err0.Log.Stacktrace, err0.Exception.Stacktrace)
}

// it should add trace attributes and report an two errors
func TestHandlerMultiErrorWithContext(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()
	var buf bytes.Buffer
	h := newApmslogHandler(&buf, tracer)
	logger := slog.New(h)

	// create transactions and spans
	tx := tracer.StartTransaction("name", "type")
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	span, ctx := apm.StartSpan(ctx, "name", "type")

	logger.ErrorContext(ctx, "hello world", "error", mockFuncError(), "err", errors.New("new error"))

	span.End()
	tx.End()

	var msg MsgWithTrace
	err := json.Unmarshal([]byte(buf.String()), &msg)
	assert.NoError(t, err)

	// assert that we added the correct traces to buff
	assert.Equal(t, tx.TraceContext().Trace.String(), msg.TraceId)
	assert.Equal(t, span.TraceContext().Span.String(), msg.SpanId)
	assert.Equal(t, tx.TraceContext().Span.String(), msg.TransactionId)

	// assert msg is as expected
	assert.Equal(t,
		fmt.Sprintf(`{"time":"1970-01-01T00:00:00Z","level":"ERROR","msg":"hello world","error":"new error","err":"new error","trace.id":"%s","transaction.id":"%s","span.id":"%s"}`,
			msg.TraceId,
			msg.TransactionId,
			msg.SpanId,
		)+"\n",
		buf.String(),
	)

	// assert we reported one error with traces to apm
	tracer.Flush(nil)
	payloads := transport.Payloads()
	assert.Len(t, payloads.Transactions, 1)
	assert.Len(t, payloads.Spans, 1)
	assert.Len(t, payloads.Errors, 2)

	// assert first apm error fields are correct
	err0 := payloads.Errors[0]
	assert.Equal(t, payloads.Spans[0].ID, err0.ParentID)
	assert.Equal(t, payloads.Transactions[0].TraceID, err0.TraceID)
	assert.Equal(t, payloads.Transactions[0].ID, err0.TransactionID)
	assert.Equal(t, "new error", err0.Exception.Message)
	assert.Equal(t, "hello world", err0.Log.Message)
	assert.Equal(t, "mockFuncError", err0.Culprit)
	assert.Equal(t, "mockFuncError", err0.Exception.Stacktrace[0].Function)
	assert.Equal(t, "(*Logger).log", err0.Log.Stacktrace[0].Function)
	assert.Equal(t, "error", err0.Log.Level)
	assert.Equal(t, "", err0.Log.LoggerName)
	assert.Equal(t, "", err0.Log.ParamMessage)
	assert.NotEmpty(t, err0.Log.Stacktrace)
	assert.NotEmpty(t, err0.Exception.Stacktrace)
	assert.NotEqual(t, err0.Log.Stacktrace, err0.Exception.Stacktrace)

	// assert second apm error fields are correct
	err1 := payloads.Errors[1]
	assert.Equal(t, payloads.Spans[0].ID, err1.ParentID)
	assert.Equal(t, payloads.Transactions[0].TraceID, err1.TraceID)
	assert.Equal(t, payloads.Transactions[0].ID, err1.TransactionID)
	assert.Equal(t, "new error", err1.Exception.Message)
	assert.Equal(t, "hello world", err1.Log.Message)
	assert.Equal(t, "TestHandlerMultiErrorWithContext", err1.Culprit)
	assert.Equal(t, "TestHandlerMultiErrorWithContext", err1.Exception.Stacktrace[0].Function)
	assert.Equal(t, "(*Logger).log", err1.Log.Stacktrace[0].Function)
	assert.Equal(t, "error", err1.Log.Level)
	assert.Equal(t, "", err1.Log.LoggerName)
	assert.Equal(t, "", err1.Log.ParamMessage)
	assert.NotEmpty(t, err1.Log.Stacktrace)
	assert.NotEmpty(t, err1.Exception.Stacktrace)
	assert.NotEqual(t, err1.Log.Stacktrace, err1.Exception.Stacktrace)
}

func TestHandlerCustomErrorAttrNoContext(t *testing.T) {
	// create apmslog handler with custom error record attributes to record
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()
	var buf bytes.Buffer
	apmHandler := apmslog.NewApmHandler(
		apmslog.WithTracer(tracer),
		apmslog.WithErrorRecordAttrs([]string{"http_error"}),
		apmslog.WithHandler(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
			Level: slog.LevelInfo,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					a.Value = slog.TimeValue(time.Unix(0, 0).UTC())
				}
				return a
			},
		})),
	)
	logger := slog.New(apmHandler)

	logger.Error("hello world", "http_error", mockFuncError(), "do_not_report_error", errors.New("do not report me"))

	// assert msg looks as expected
	assert.Equal(t, `{"time":"1970-01-01T00:00:00Z","level":"ERROR","msg":"hello world","http_error":"new error","do_not_report_error":"do not report me"}`+"\n", buf.String())

	// assert we reported one error with traces to apm
	tracer.Flush(nil)
	payloads := transport.Payloads()
	assert.Empty(t, payloads.Transactions, 0)
	assert.Empty(t, payloads.Spans, 0)
	assert.Len(t, payloads.Errors, 1)

	// assert apm error fields are correct
	err0 := payloads.Errors[0]
	assert.Equal(t, "new error", err0.Exception.Message)
	assert.Equal(t, "hello world", err0.Log.Message)
	assert.Equal(t, "mockFuncError", err0.Culprit)
	assert.Equal(t, "mockFuncError", err0.Exception.Stacktrace[0].Function)
	assert.Equal(t, "(*Logger).log", err0.Log.Stacktrace[0].Function)
	assert.Equal(t, "error", err0.Log.Level)
	assert.Equal(t, "", err0.Log.LoggerName)
	assert.Equal(t, "", err0.Log.ParamMessage)
	assert.NotEmpty(t, err0.Log.Stacktrace)
	assert.NotEmpty(t, err0.Exception.Stacktrace)
	assert.NotEqual(t, err0.Log.Stacktrace, err0.Exception.Stacktrace)
}

// assert that logger still works if tracer is closed
func TestHandlerTracerClosed(t *testing.T) {
	tracer, _ := transporttest.NewRecorderTracer()
	tracer.Close()

	var buf bytes.Buffer

	h := newApmslogHandler(&buf, tracer)
	logger := slog.New(h)
	logger.Error("hello world")
	assert.Equal(t, `{"time":"1970-01-01T00:00:00Z","level":"ERROR","msg":"hello world"}`+"\n", buf.String())
}

func newApmslogHandler(writer io.Writer, tracer *apm.Tracer) *apmslog.ApmHandler {
	apmHandler := apmslog.NewApmHandler(
		apmslog.WithTracer(tracer),
		apmslog.WithHandler(slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level: slog.LevelInfo,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					a.Value = slog.TimeValue(time.Unix(0, 0).UTC())
				}
				return a
			},
		})),
	)
	return apmHandler
}
