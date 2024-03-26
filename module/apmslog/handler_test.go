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

func TestHandler(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()
	var buf bytes.Buffer

	h := newApmslogHandler(&buf, tracer)
	logger := slog.New(h)

	logger.Error("hello world", "error", errors.New("new error"))

	assert.Equal(t, `{"time":"1970-01-01T00:00:00Z","level":"ERROR","msg":"hello world","error":"new error"}`+"\n", buf.String())

	tracer.Flush(nil)
	payloads := transport.Payloads()
	assert.Len(t, payloads.Errors, 1)

	err0 := payloads.Errors[0]
	assert.Equal(t, "hello world", err0.Log.Message)
	assert.Equal(t, "error", err0.Log.Level)
	assert.Equal(t, "", err0.Log.LoggerName)
	assert.Equal(t, "", err0.Log.ParamMessage)
	assert.Equal(t, "TestHandler", err0.Culprit)
	assert.NotEmpty(t, err0.Log.Stacktrace)
	// assert.Equal(t, model.Time(time.Unix(0, 0).UTC()), err0.Timestamp) // seems like slog time attribute is not accessable
	assert.Zero(t, err0.ParentID)
	assert.Zero(t, err0.TraceID)
	assert.Zero(t, err0.TransactionID)
}
func TestHandlerTransactionTraceContext(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	var buf bytes.Buffer

	h := newApmslogHandler(&buf, tracer)
	logger := slog.New(h)

	tx := tracer.StartTransaction("name", "type")
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	span, ctx := apm.StartSpan(ctx, "name", "type")

	logger.ErrorContext(ctx, "hello world", "error", errors.New("new error"))

	span.End()
	tx.End()

	tracer.Flush(nil)
	payloads := transport.Payloads()
	assert.Len(t, payloads.Transactions, 1)
	assert.Len(t, payloads.Spans, 1)
	assert.Len(t, payloads.Errors, 1)

	err0 := payloads.Errors[0]
	assert.Equal(t, payloads.Spans[0].ID, err0.ParentID)
	assert.Equal(t, payloads.Transactions[0].TraceID, err0.TraceID)
	assert.Equal(t, payloads.Transactions[0].ID, err0.TransactionID)
}

func TestHandlerWithError(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	var buf bytes.Buffer

	h := newApmslogHandler(&buf, tracer)
	logger := slog.New(h)

	logger.Error("hello world", "error", mockFuncError())

	tracer.Flush(nil)
	payloads := transport.Payloads()
	assert.Len(t, payloads.Errors, 1)

	err0 := payloads.Errors[0]
	assert.Equal(t, "new error", err0.Exception.Message)
	assert.Equal(t, "hello world", err0.Log.Message)
	assert.Equal(t, "mockFuncError", err0.Culprit)
	assert.NotEmpty(t, err0.Log.Stacktrace)
	assert.NotEmpty(t, err0.Exception.Stacktrace)
	assert.NotEqual(t, err0.Log.Stacktrace, err0.Exception.Stacktrace)
	assert.Equal(t, "mockFuncError", err0.Exception.Stacktrace[0].Function)
	assert.Equal(t, "(*Logger).log", err0.Log.Stacktrace[0].Function)
}

func mockFuncError() error {
	return errors.New("new error")
}

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
