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

package apmzap_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmzap"
	"go.elastic.co/apm/transport/transporttest"
)

func TestCore(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	core := &apmzap.Core{Tracer: tracer}
	logger := zap.New(core).Named("myLogger")
	logger.Error("¡hola, mundo!")

	tracer.Flush(nil)
	payloads := transport.Payloads()
	assert.Len(t, payloads.Errors, 1)

	err0 := payloads.Errors[0]
	assert.Equal(t, "¡hola, mundo!", err0.Log.Message)
	assert.Equal(t, "error", err0.Log.Level)
	assert.Equal(t, "myLogger", err0.Log.LoggerName)
	assert.Equal(t, "", err0.Log.ParamMessage)
	assert.Equal(t, "TestCore", err0.Culprit)
	assert.NotEmpty(t, err0.Log.Stacktrace)
	assert.Zero(t, err0.ParentID)
	assert.Zero(t, err0.TraceID)
	assert.Zero(t, err0.TransactionID)
}

func TestCoreTransactionTraceContext(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	core := &apmzap.Core{Tracer: tracer}
	logger := zap.New(core).Named("myLogger")

	tx := tracer.StartTransaction("name", "type")
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	span, ctx := apm.StartSpan(ctx, "name", "type")
	logger.With(apmzap.TraceContext(ctx)...).Error("¡hola, mundo!")
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

func TestCoreWithError(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	core := &apmzap.Core{Tracer: tracer}
	logger := zap.New(core)
	logger.Error("nope nope nope", zap.Error(makeError()))

	tracer.Flush(nil)
	payloads := transport.Payloads()
	assert.Len(t, payloads.Errors, 1)

	err0 := payloads.Errors[0]
	assert.Equal(t, "kablamo", err0.Exception.Message)
	assert.Equal(t, "nope nope nope", err0.Log.Message)
	assert.Equal(t, "makeError", err0.Culprit)
	assert.NotEmpty(t, err0.Log.Stacktrace)
	assert.NotEmpty(t, err0.Exception.Stacktrace)
	assert.NotEqual(t, err0.Log.Stacktrace, err0.Exception.Stacktrace)
	assert.Equal(t, "makeError", err0.Exception.Stacktrace[0].Function)
	assert.Equal(t, "(*contextCore).Write", err0.Log.Stacktrace[0].Function)
}

func makeError() error {
	return errors.New("kablamo")
}

func TestCoreTracerClosed(t *testing.T) {
	tracer, _ := transporttest.NewRecorderTracer()
	tracer.Close() // close it straight away, core should return immediately

	core := &apmzap.Core{Tracer: tracer}
	logger := zap.New(core)
	logger.Error("boom")
}

func TestCoreFatal(t *testing.T) {
	if os.Getenv("_INSIDE_TEST") == "1" {
		tracer, _ := apm.NewTracer("", "")
		logger := zap.New(&apmzap.Core{Tracer: tracer})
		logger.Fatal("fatality!")
	}

	var recorder transporttest.RecorderTransport
	mux := http.NewServeMux()
	mux.HandleFunc("/intake/v2/events", func(w http.ResponseWriter, req *http.Request) {
		if err := recorder.SendStream(req.Context(), req.Body); err != nil {
			panic(err)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	cmd := exec.Command(os.Args[0], "-test.run=^TestCoreFatal$")
	cmd.Env = append(os.Environ(),
		"_INSIDE_TEST=1",
		"ELASTIC_APM_SERVER_URL="+server.URL,
		"ELASTIC_APM_LOG_FILE=stderr",
		"ELASTIC_APM_LOG_LEVEL=debug",
	)
	output, err := cmd.CombinedOutput()
	assert.Error(t, err)
	defer func() {
		if t.Failed() {
			t.Logf("%s", output)
		}
	}()

	payloads := recorder.Payloads()
	require.Len(t, payloads.Errors, 1)
	assert.Equal(t, "fatality!", payloads.Errors[0].Log.Message)
	assert.Equal(t, "fatal", payloads.Errors[0].Log.Level)
}
