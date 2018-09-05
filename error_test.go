package elasticapm_test

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/stacktrace"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

func TestErrorsStackTrace(t *testing.T) {
	modelError := sendError(t, &errorsStackTracer{
		"zing", newErrorsStackTrace(0, 2),
	})
	exception := modelError.Exception
	stacktrace := exception.Stacktrace
	assert.Equal(t, "zing", exception.Message)
	assert.Equal(t, "github.com/elastic/apm-agent-go_test", exception.Module)
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
	assert.Equal(t, "github.com/elastic/apm-agent-go_test", exception.Module)
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
	assert.NotEmpty(t, payloads)
	for _, p := range payloads {
		errors := p.Errors()
		assert.NotEmpty(t, errors)
		for _, e := range errors {
			assert.NotEqual(t, "", e.Culprit)
			assert.NotEmpty(t, e.Exception.Stacktrace)
		}
	}
}

func sendError(t *testing.T, err error, f ...func(*elasticapm.Error)) *model.Error {
	tracer, r := transporttest.NewRecorderTracer()
	defer tracer.Close()

	error_ := tracer.NewError(err)
	for _, f := range f {
		f(error_)
	}

	error_.Send()
	tracer.Flush(nil)

	payloads := r.Payloads()
	require.Len(t, payloads, 1)
	errors := payloads[0].Errors()
	require.Len(t, errors, 1)
	return errors[0]
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
	frames := make([]errors.Frame, len(callers))
	for i, pc := range callers {
		frames[i] = errors.Frame(pc)
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
