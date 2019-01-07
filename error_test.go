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

package apm_test

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/stacktrace"
	"go.elastic.co/apm/transport/transporttest"
)

func TestErrorID(t *testing.T) {
	var errorID apm.ErrorID
	_, _, errors := apmtest.WithTransaction(func(ctx context.Context) {
		e := apm.CaptureError(ctx, errors.New("boom"))
		errorID = e.ID
		e.Send()
	})
	require.Len(t, errors, 1)
	assert.NotZero(t, errorID)
	assert.Equal(t, model.TraceID(errorID), errors[0].ID)
}

func TestErrorsStackTrace(t *testing.T) {
	modelError := sendError(t, &errorsStackTracer{
		"zing", newErrorsStackTrace(0, 2),
	})
	exception := modelError.Exception
	stacktrace := exception.Stacktrace
	assert.Equal(t, "zing", exception.Message)
	assert.Equal(t, "go.elastic.co/apm_test", exception.Module)
	assert.Equal(t, "errorsStackTracer", exception.Type)
	assert.Len(t, stacktrace, 2)
	assert.Equal(t, "newErrorsStackTrace", stacktrace[0].Function)
	assert.Equal(t, "TestErrorsStackTrace", stacktrace[1].Function)
}

func TestInternalStackTrace(t *testing.T) {
	// Absolute path on both windows (UNC) and *nix
	abspath := filepath.FromSlash("//abs/path/file.go")
	modelError := sendError(t, &internalStackTracer{
		"zing", []stacktrace.Frame{
			{Function: "pkg/path.FuncName"},
			{Function: "FuncName2", File: abspath, Line: 123},
			{Function: "encoding/json.Marshal"},
		},
	})
	exception := modelError.Exception
	stacktrace := exception.Stacktrace
	assert.Equal(t, "zing", exception.Message)
	assert.Equal(t, "go.elastic.co/apm_test", exception.Module)
	assert.Equal(t, "internalStackTracer", exception.Type)
	assert.Equal(t, []model.StacktraceFrame{{
		Function: "FuncName",
		Module:   "pkg/path",
	}, {
		AbsolutePath: abspath,
		Function:     "FuncName2",
		File:         "file.go",
		Line:         123,
	}, {
		Function:     "Marshal",
		Module:       "encoding/json",
		LibraryFrame: true,
	}}, stacktrace)
}

func TestErrorAutoStackTraceReuse(t *testing.T) {
	tracer, r := transporttest.NewRecorderTracer()
	defer tracer.Close()

	err := fmt.Errorf("hullo") // no stacktrace attached
	for i := 0; i < 1000; i++ {
		tracer.NewError(err).Send()
	}
	tracer.Flush(nil)

	// The previously sent error objects should have
	// been reset and will be reused. We reuse the
	// stacktrace slice. See elastic/apm-agent-go#204.
	for i := 0; i < 1000; i++ {
		tracer.NewError(err).Send()
	}
	tracer.Flush(nil)

	payloads := r.Payloads()
	assert.NotEmpty(t, payloads.Errors)
	for _, e := range payloads.Errors {
		assert.NotEqual(t, "", e.Culprit)
		assert.NotEmpty(t, e.Exception.Stacktrace)
	}
}

func TestCaptureErrorNoTransaction(t *testing.T) {
	// When there's no transaction or span in the context,
	// CaptureError returns nil as it has no tracer with
	// which it can create the error.
	e := apm.CaptureError(context.Background(), errors.New("boom"))
	assert.Nil(t, e)

	// Send is a no-op on a nil Error.
	e.Send()
}

func TestErrorLogRecord(t *testing.T) {
	tracer, recorder := transporttest.NewRecorderTracer()
	defer tracer.Close()

	error_ := tracer.NewErrorLog(apm.ErrorLogRecord{
		Message: "log-message",
		Error:   makeError("error-message"),
	})
	error_.SetStacktrace(1)
	error_.Send()
	tracer.Flush(nil)

	payloads := recorder.Payloads()
	require.Len(t, payloads.Errors, 1)
	err0 := payloads.Errors[0]
	assert.Equal(t, "log-message", err0.Log.Message)
	assert.Equal(t, "error-message", err0.Exception.Message)
	require.NotEmpty(t, err0.Log.Stacktrace)
	require.NotEmpty(t, err0.Exception.Stacktrace)
	assert.Equal(t, err0.Log.Stacktrace[0].Function, "TestErrorLogRecord")
	assert.Equal(t, err0.Exception.Stacktrace[0].Function, "makeError")
	assert.Equal(t, "makeError", err0.Culprit) // based on exception stacktrace
}

func TestErrorTransactionSampled(t *testing.T) {
	_, _, errors := apmtest.WithTransaction(func(ctx context.Context) {
		apm.CaptureError(ctx, errors.New("boom")).Send()

		span, ctx := apm.StartSpan(ctx, "name", "type")
		defer span.End()
		apm.CaptureError(ctx, errors.New("boom")).Send()
	})
	assertErrorTransactionSampled(t, errors[0], true)
	assertErrorTransactionSampled(t, errors[1], true)
}

func TestErrorTransactionNotSampled(t *testing.T) {
	tracer, recorder := transporttest.NewRecorderTracer()
	defer tracer.Close()
	tracer.SetSampler(apm.NewRatioSampler(0))

	tx := tracer.StartTransaction("name", "type")
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	apm.CaptureError(ctx, errors.New("boom")).Send()

	tracer.Flush(nil)
	payloads := recorder.Payloads()
	require.Len(t, payloads.Errors, 1)
	assertErrorTransactionSampled(t, payloads.Errors[0], false)
}

func TestErrorTransactionSampledNoTransaction(t *testing.T) {
	tracer, recorder := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.NewError(errors.New("boom")).Send()
	tracer.Flush(nil)
	payloads := recorder.Payloads()
	require.Len(t, payloads.Errors, 1)
	assert.Nil(t, payloads.Errors[0].Transaction.Sampled)
}

func assertErrorTransactionSampled(t *testing.T, e model.Error, sampled bool) {
	assert.Equal(t, &sampled, e.Transaction.Sampled)
}

func makeError(msg string) error {
	return errors.New(msg)
}

func sendError(t *testing.T, err error, f ...func(*apm.Error)) model.Error {
	tracer, r := transporttest.NewRecorderTracer()
	defer tracer.Close()

	error_ := tracer.NewError(err)
	for _, f := range f {
		f(error_)
	}

	error_.Send()
	tracer.Flush(nil)

	payloads := r.Payloads()
	return payloads.Errors[0]
}

type errorsStackTracer struct {
	message    string
	stackTrace errors.StackTrace
}

func (e *errorsStackTracer) Error() string {
	return e.message
}

func (e *errorsStackTracer) StackTrace() errors.StackTrace {
	return e.stackTrace
}

func newErrorsStackTrace(skip, n int) errors.StackTrace {
	callers := make([]uintptr, 2)
	callers = callers[:runtime.Callers(1, callers)]

	var (
		uintptrType      = reflect.TypeOf(uintptr(0))
		errorsFrameType  = reflect.TypeOf(*new(errors.Frame))
		runtimeFrameType = reflect.TypeOf(runtime.Frame{})
	)

	var frames []errors.Frame
	switch {
	case errorsFrameType.ConvertibleTo(uintptrType):
		frames = make([]errors.Frame, len(callers))
		for i, pc := range callers {
			reflect.ValueOf(&frames[i]).Elem().Set(reflect.ValueOf(pc).Convert(errorsFrameType))
		}
	case errorsFrameType.ConvertibleTo(runtimeFrameType):
		fs := runtime.CallersFrames(callers)
		for {
			var frame errors.Frame
			runtimeFrame, more := fs.Next()
			reflect.ValueOf(&frame).Elem().Set(reflect.ValueOf(runtimeFrame).Convert(errorsFrameType))
			frames = append(frames, frame)
			if !more {
				break
			}
		}
	default:
		panic(fmt.Errorf("unhandled errors.Frame type %s", errorsFrameType))
	}
	return errors.StackTrace(frames)
}

type internalStackTracer struct {
	message string
	frames  []stacktrace.Frame
}

func (e *internalStackTracer) Error() string {
	return e.message
}

func (e *internalStackTracer) StackTrace() []stacktrace.Frame {
	return e.frames
}
