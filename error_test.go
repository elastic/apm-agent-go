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
	"net"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"syscall"
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
	require.Len(t, stacktrace, 2)
	assert.Equal(t, "newErrorsStackTrace", stacktrace[0].Function)
	assert.Equal(t, "TestErrorsStackTrace", stacktrace[1].Function)
}

func TestErrorsStackTraceLimit(t *testing.T) {
	defer os.Unsetenv("ELASTIC_APM_STACK_TRACE_LIMIT")
	const n = 2
	for i := -1; i < n; i++ {
		os.Setenv("ELASTIC_APM_STACK_TRACE_LIMIT", strconv.Itoa(i))
		modelError := sendError(t, &errorsStackTracer{
			"zing", newErrorsStackTrace(0, n),
		})
		stacktrace := modelError.Exception.Stacktrace
		if i == -1 {
			require.Len(t, stacktrace, n)
		} else {
			require.Len(t, stacktrace, i)
		}
	}
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

func TestInternalStackTraceLimit(t *testing.T) {
	inFrames := []stacktrace.Frame{
		{Function: "pkg/path.FuncName"},
		{Function: "FuncName2", Line: 123},
		{Function: "encoding/json.Marshal"},
	}
	outFrames := []model.StacktraceFrame{{
		Function: "FuncName",
		Module:   "pkg/path",
	}, {
		Function: "FuncName2",
		Line:     123,
	}, {
		Function:     "Marshal",
		Module:       "encoding/json",
		LibraryFrame: true,
	}}

	defer os.Unsetenv("ELASTIC_APM_STACK_TRACE_LIMIT")
	for i := -1; i < len(inFrames); i++ {
		os.Setenv("ELASTIC_APM_STACK_TRACE_LIMIT", strconv.Itoa(i))
		modelError := sendError(t, &internalStackTracer{
			"zing", []stacktrace.Frame{
				{Function: "pkg/path.FuncName"},
				{Function: "FuncName2", Line: 123},
				{Function: "encoding/json.Marshal"},
			},
		})
		stacktrace := modelError.Exception.Stacktrace
		if i == 0 {
			assert.Nil(t, stacktrace)
			continue
		}
		expect := outFrames
		if i > 0 {
			expect = expect[:i]
		}
		assert.Equal(t, expect, stacktrace)
	}
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
	// CaptureError returns Error with nil ErrorData as it has no tracer with
	// which it can create the error.
	e := apm.CaptureError(context.Background(), errors.New("boom"))
	assert.Nil(t, e.ErrorData)

	// Send is a no-op on a Error with nil ErrorData.
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

func TestErrorCauserInterface(t *testing.T) {
	type Causer interface {
		Cause() error
	}
	var e Causer = apm.CaptureError(context.Background(), errors.New("boom"))
	assert.EqualError(t, e.Cause(), "boom")
}

func TestErrorNilCauser(t *testing.T) {
	var e *apm.Error
	assert.Nil(t, e.Cause())

	e = &apm.Error{}
	assert.Nil(t, e.Cause())
}

func TestErrorErrorInterface(t *testing.T) {
	var e error = apm.CaptureError(context.Background(), errors.New("boom"))
	assert.EqualError(t, e, "boom")
}

func TestErrorNilError(t *testing.T) {
	var e *apm.Error
	assert.EqualError(t, e, "[EMPTY]")

	e = &apm.Error{}
	assert.EqualError(t, e, "")
}

func TestErrorTransactionSampled(t *testing.T) {
	_, _, errors := apmtest.WithTransaction(func(ctx context.Context) {
		apm.TransactionFromContext(ctx).Type = "foo"
		apm.CaptureError(ctx, errors.New("boom")).Send()

		span, ctx := apm.StartSpan(ctx, "name", "type")
		defer span.End()
		apm.CaptureError(ctx, errors.New("boom")).Send()
	})
	assertErrorTransactionSampled(t, errors[0], true)
	assertErrorTransactionSampled(t, errors[1], true)
	assert.Equal(t, "foo", errors[0].Transaction.Type)
	assert.Equal(t, "foo", errors[1].Transaction.Type)
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

func TestErrorTransactionCustomContext(t *testing.T) {
	tracer, recorder := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tx := tracer.StartTransaction("name", "type")
	tx.Context.SetCustom("k1", "v1")
	tx.Context.SetCustom("k2", "v2")
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	apm.CaptureError(ctx, errors.New("boom")).Send()

	_, ctx = apm.StartSpan(ctx, "foo", "bar")
	apm.CaptureError(ctx, errors.New("boom")).Send()

	// Create an error with custom context set before setting
	// the transaction. Such custom context should override
	// whatever is carried over from the transaction.
	e := tracer.NewError(errors.New("boom"))
	e.Context.SetCustom("k1", "!!")
	e.Context.SetCustom("k3", "v3")
	e.SetTransaction(tx)
	e.Send()

	tracer.Flush(nil)
	payloads := recorder.Payloads()
	require.Len(t, payloads.Errors, 3)

	assert.Equal(t, model.IfaceMap{
		{Key: "k1", Value: "v1"},
		{Key: "k2", Value: "v2"},
	}, payloads.Errors[0].Context.Custom)

	assert.Equal(t, model.IfaceMap{
		{Key: "k1", Value: "v1"},
		{Key: "k2", Value: "v2"},
	}, payloads.Errors[1].Context.Custom)

	assert.Equal(t, model.IfaceMap{
		{Key: "k1", Value: "!!"},
		{Key: "k2", Value: "v2"},
		{Key: "k3", Value: "v3"},
	}, payloads.Errors[2].Context.Custom)
}

func TestErrorDetailer(t *testing.T) {
	type error1 struct{ error }
	apm.RegisterTypeErrorDetailer(reflect.TypeOf(error1{}), apm.ErrorDetailerFunc(func(err error, details *apm.ErrorDetails) {
		details.SetAttr("a", "error1")
	}))

	type error2 struct{ error }
	apm.RegisterTypeErrorDetailer(reflect.TypeOf(&error2{}), apm.ErrorDetailerFunc(func(err error, details *apm.ErrorDetails) {
		details.SetAttr("b", "*error2")
	}))

	apm.RegisterErrorDetailer(apm.ErrorDetailerFunc(func(err error, details *apm.ErrorDetails) {
		// NOTE(axw) ErrorDetailers can't be _unregistered_,
		// so we check the error type so as not to interfere
		// with other tests.
		switch err.(type) {
		case error1, *error2:
			details.SetAttr("c", "both")
		}
	}))

	_, _, errs := apmtest.WithTransaction(func(ctx context.Context) {
		apm.CaptureError(ctx, error1{errors.New("error1")}).Send()
		apm.CaptureError(ctx, &error2{errors.New("error2")}).Send()
	})
	require.Len(t, errs, 2)
	assert.Equal(t, map[string]interface{}{"a": "error1", "c": "both"}, errs[0].Exception.Attributes)
	assert.Equal(t, map[string]interface{}{"b": "*error2", "c": "both"}, errs[1].Exception.Attributes)
}

func TestStdlibErrorDetailers(t *testing.T) {
	t.Run("syscall.Errno", func(t *testing.T) {
		_, _, errs := apmtest.WithTransaction(func(ctx context.Context) {
			apm.CaptureError(ctx, syscall.Errno(syscall.EAGAIN)).Send()
		})
		require.Len(t, errs, 1)

		if runtime.GOOS == "windows" {
			// There's currently no equivalent of unix.ErrnoName for Windows.
			assert.Equal(t, model.ExceptionCode{Number: float64(syscall.EAGAIN)}, errs[0].Exception.Code)
		} else {
			assert.Equal(t, model.ExceptionCode{String: "EAGAIN"}, errs[0].Exception.Code)
		}

		assert.Equal(t, map[string]interface{}{
			"temporary": true,
			"timeout":   true,
		}, errs[0].Exception.Attributes)
	})

	cause := errors.New("cause")
	test := func(err error, expectedAttrs map[string]interface{}) {
		t.Run(fmt.Sprintf("%T", err), func(t *testing.T) {
			_, _, errs := apmtest.WithTransaction(func(ctx context.Context) {
				apm.CaptureError(ctx, err).Send()
			})
			require.Len(t, errs, 1)
			assert.Equal(t, expectedAttrs, errs[0].Exception.Attributes)
			require.Len(t, errs[0].Exception.Cause, 1)
			assert.Equal(t, "cause", errs[0].Exception.Cause[0].Message)
		})
	}
	type attrmap map[string]interface{}

	test(&net.OpError{
		Err: cause,
		Op:  "read",
		Net: "tcp",
		Source: &net.TCPAddr{
			IP:   net.IPv6loopback,
			Port: 1234,
		},
	}, attrmap{"op": "read", "net": "tcp", "source": "tcp:[::1]:1234"})

	test(&os.LinkError{
		Err: cause,
		Op:  "symlink",
		Old: "/old",
		New: "/new",
	}, attrmap{"op": "symlink", "old": "/old", "new": "/new"})

	test(&os.PathError{
		Err:  cause,
		Op:   "open",
		Path: "/dev/null",
	}, attrmap{"op": "open", "path": "/dev/null"})

	test(&os.SyscallError{
		Err:     cause,
		Syscall: "connect",
	}, attrmap{"syscall": "connect"})
}

func TestErrorCauseCulprit(t *testing.T) {
	err := errors.WithStack(testErrorCauseCulpritHelper())

	tracer, recorder := transporttest.NewRecorderTracer()
	defer tracer.Close()
	tracer.NewError(err).Send()
	tracer.Flush(nil)

	payloads := recorder.Payloads()
	require.Len(t, payloads.Errors, 1)
	assert.Equal(t, "testErrorCauseCulpritHelper", payloads.Errors[0].Culprit)
}

func testErrorCauseCulpritHelper() error {
	return errors.Errorf("something happened here")
}

func TestErrorCauseCauser(t *testing.T) {
	err := &causer{
		error: errors.New("error"),
		cause: errors.New("cause"),
	}

	tracer, recorder := transporttest.NewRecorderTracer()
	defer tracer.Close()
	tracer.NewError(err).Send()
	tracer.Flush(nil)

	payloads := recorder.Payloads()
	require.Len(t, payloads.Errors, 1)
	assert.Equal(t, "TestErrorCauseCauser", payloads.Errors[0].Culprit)

	require.Len(t, payloads.Errors[0].Exception.Cause, 1)
	assert.Equal(t, "cause", payloads.Errors[0].Exception.Cause[0].Message)
}

func TestErrorCauseCycle(t *testing.T) {
	err := make(errorslice, 1)
	err[0] = causer{error: makeError("error"), cause: &err}

	tracer, recorder := transporttest.NewRecorderTracer()
	defer tracer.Close()
	tracer.NewError(err).Send()
	tracer.Flush(nil)

	payloads := recorder.Payloads()
	require.Len(t, payloads.Errors, 1)
	assert.Equal(t, "TestErrorCauseCycle", payloads.Errors[0].Culprit)

	require.Len(t, payloads.Errors[0].Exception.Cause, 1)
	require.Len(t, payloads.Errors[0].Exception.Cause[0].Cause, 1)
	require.Len(t, payloads.Errors[0].Exception.Cause[0].Cause, 1) // cycle broken
	assert.Equal(t, "error", payloads.Errors[0].Exception.Cause[0].Message)
	assert.Equal(t, "errorslice", payloads.Errors[0].Exception.Cause[0].Cause[0].Message)
}

func assertErrorTransactionSampled(t *testing.T, e model.Error, sampled bool) {
	assert.Equal(t, &sampled, e.Transaction.Sampled)
	if sampled {
		assert.NotEmpty(t, e.Transaction.Type)
	} else {
		assert.Empty(t, e.Transaction.Type)
	}
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

type causer struct {
	error
	cause error
}

func (c causer) Cause() error {
	return c.cause
}

type errorslice []error

func (es errorslice) Error() string {
	return "errorslice"
}

func (es errorslice) Cause() error {
	return es[0]
}
