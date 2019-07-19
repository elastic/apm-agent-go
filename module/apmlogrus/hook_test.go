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

package apmlogrus_test

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmlogrus"
	"go.elastic.co/apm/transport/transporttest"
)

func TestHook(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	var buf bytes.Buffer
	logger := newLogger(&buf)
	logger.AddHook(&apmlogrus.Hook{Tracer: tracer})
	logger.WithTime(time.Unix(0, 0).UTC()).Errorf("¡hola, %s!", "mundo")

	assert.Equal(t, `{"level":"error","msg":"¡hola, mundo!","time":"1970-01-01T00:00:00Z"}`+"\n", buf.String())

	tracer.Flush(nil)
	payloads := transport.Payloads()
	assert.Len(t, payloads.Errors, 1)

	err0 := payloads.Errors[0]
	assert.Equal(t, "¡hola, mundo!", err0.Log.Message)
	assert.Equal(t, "error", err0.Log.Level)
	assert.Equal(t, "", err0.Log.LoggerName)
	assert.Equal(t, "", err0.Log.ParamMessage)
	assert.Equal(t, "TestHook", err0.Culprit)
	assert.NotEmpty(t, err0.Log.Stacktrace)
	assert.Equal(t, model.Time(time.Unix(0, 0).UTC()), err0.Timestamp)
	assert.Zero(t, err0.ParentID)
	assert.Zero(t, err0.TraceID)
	assert.Zero(t, err0.TransactionID)
}

func TestHookTransactionTraceContext(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	logger := newLogger(ioutil.Discard)
	logger.AddHook(&apmlogrus.Hook{Tracer: tracer})

	tx := tracer.StartTransaction("name", "type")
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	span, ctx := apm.StartSpan(ctx, "name", "type")
	logger.WithFields(apmlogrus.TraceContext(ctx)).Errorf("¡hola, %s!", "mundo")
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

func TestHookWithError(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	logger := newLogger(ioutil.Discard)
	logger.AddHook(&apmlogrus.Hook{Tracer: tracer})
	logger.WithError(makeError()).Error("nope nope nope")

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
	assert.Equal(t, "(*Hook).Fire", err0.Log.Stacktrace[0].Function)
}

func makeError() error {
	return errors.New("kablamo")
}

func TestHookFatal(t *testing.T) {
	if os.Getenv("_INSIDE_TEST") == "1" {
		tracer, _ := apm.NewTracer("", "")
		logger := logrus.New()
		logger.AddHook(&apmlogrus.Hook{Tracer: tracer})
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

	cmd := exec.Command(os.Args[0], "-test.run=^TestHookFatal$")
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

func TestHookTracerClosed(t *testing.T) {
	tracer, _ := transporttest.NewRecorderTracer()
	tracer.Close() // close it straight away, hook should return immediately

	logger := newLogger(ioutil.Discard)
	logger.AddHook(&apmlogrus.Hook{Tracer: tracer})
	logger.Error("boom")
}

func newLogger(w io.Writer) *logrus.Logger {
	return &logrus.Logger{
		Out:       w,
		Formatter: new(logrus.JSONFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}
}
