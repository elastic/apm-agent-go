package trace

import (
	"fmt"
	"net"
	"os"
	"reflect"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/stacktrace"
)

// Recover recovers panics, enqueuing exception errors.
// Recover is expected to be used in a deferred call.
//
// If the error implements
//   type stackTracer interface{
//       StackTrace() github.com/pkg/errors.StackTrace
//   }
// then that will be used to set the exception stacktrace.
// Otherwise, Recover will take a stacktrace, skipping
// this Recover call's frame.
func (t *Tracer) Recover(tx *Transaction) {
	v := recover()
	if v == nil {
		return
	}
	e := t.Recovered(v, tx)
	if e.Exception.Stacktrace == nil {
		e.SetExceptionStacktrace(1)
	}
	e.Send()
}

// Recovered creates and returns an Error with its Exception
// initialized from the recovered value v, which is expected
// to have come from a panic.
//
// If v is an error, the Exception will be initialized using
// Error.SetException, and its stacktrace may be set. Otherwise,
// Exception will be initialized with the message set to
// fmt.Sprint(v), and its stacktrace will be nil.
func (t *Tracer) Recovered(v interface{}, tx *Transaction) *Error {
	e := t.NewError()
	e.Transaction = tx
	e.ID, _ = NewUUID() // ignore error result
	switch v := v.(type) {
	case error:
		e.SetException(v)
	default:
		e.Exception = &model.Exception{
			Message: fmt.Sprint(v),
		}
	}
	return e
}

// NewError returns a new Error associated with the Tracer.
func (t *Tracer) NewError() *Error {
	e, _ := t.errorPool.Get().(*Error)
	if e == nil {
		e = &Error{tracer: t}
	}
	e.Timestamp = time.Now()
	return e
}

// Error describes an error occurring in the monitored service.
type Error struct {
	model.Error
	Transaction *Transaction
	tracer      *Tracer
}

func (e *Error) reset() {
	tracer := e.tracer
	*e = Error{}
	e.tracer = tracer
}

// Send enqueues the error for sending to the Elastic APM server.
// The Error must not be used after this.
func (e *Error) Send() {
	select {
	case e.tracer.errors <- e:
	default:
		// Enqueuing an error should never block.
		e.tracer.statsMu.Lock()
		e.tracer.stats.ErrorsDropped++
		e.tracer.statsMu.Unlock()
		e.reset()
		e.tracer.errorPool.Put(e)
	}
}

func (e *Error) setCulprit() {
	if e.Culprit == "" && e.Exception != nil {
		e.Culprit = stacktraceCulprit(e.Exception.Stacktrace)
	}
	if e.Culprit == "" && e.Log != nil {
		e.Culprit = stacktraceCulprit(e.Log.Stacktrace)
	}
}

func (e *Error) setContext(setter stacktrace.ContextSetter, pre, post int) error {
	if e.Exception != nil {
		if err := stacktrace.SetContext(setter, e.Exception.Stacktrace, pre, post); err != nil {
			return err
		}
	}
	if e.Log != nil {
		if err := stacktrace.SetContext(setter, e.Log.Stacktrace, pre, post); err != nil {
			return err
		}
	}
	return nil
}

// stacktraceCulprit returns the first non-library stacktrace frame's
// function name.
func stacktraceCulprit(frames []model.StacktraceFrame) string {
	for _, frame := range frames {
		if !frame.LibraryFrame {
			return frame.Function
		}
	}
	return ""
}

// SetException sets the Error's Exception field, based on the given error.
//
// Message will be set to err.Error(). The Module and Type fields will be set
// to the package and type name of the cause of the error, where the cause has
// the same definition as given by github.com/pkg/errors.
func (e *Error) SetException(err error) {
	if err == nil {
		panic("SetError must be called with a non-nil error")
	}
	e.Exception = &model.Exception{
		Message: err.Error(),
	}
	initException(e.Exception, errors.Cause(err))
	initErrorsStacktrace(e.Exception, err)
}

func initException(e *model.Exception, err error) {
	setAttr := func(k string, v interface{}) {
		if e.Attributes == nil {
			e.Attributes = make(map[string]interface{})
		}
		e.Attributes[k] = v
	}

	// Set Module, Type, Attributes, and Code.
	switch err := err.(type) {
	case *net.OpError:
		e.Module, e.Type = "net", "OpError"
		setAttr("op", err.Op)
		setAttr("net", err.Net)
		setAttr("source", err.Source)
		setAttr("addr", err.Addr)
	case *os.LinkError:
		e.Module, e.Type = "os", "LinkError"
		setAttr("op", err.Op)
		setAttr("old", err.Old)
		setAttr("new", err.New)
	case *os.PathError:
		e.Module, e.Type = "os", "PathError"
		setAttr("op", err.Op)
		setAttr("path", err.Path)
	case *os.SyscallError:
		e.Module, e.Type = "os", "SyscallError"
		setAttr("syscall", err.Syscall)
	case syscall.Errno:
		e.Module, e.Type = "syscall", "Errno"
		e.Code = uintptr(err)
	default:
		t := reflect.TypeOf(err)
		if t.Name() == "" && t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		e.Module, e.Type = t.PkgPath(), t.Name()
	}
	if errTemporary(err) {
		setAttr("temporary", true)
	}
	if errTimeout(err) {
		setAttr("timeout", true)
	}
}

func initErrorsStacktrace(e *model.Exception, err error) {
	type stackTracer interface {
		StackTrace() errors.StackTrace
	}
	if stackTracer, ok := err.(stackTracer); ok {
		stackTrace := stackTracer.StackTrace()
		pc := make([]uintptr, len(stackTrace))
		for i, frame := range stackTrace {
			pc[i] = uintptr(frame)
		}
		e.Stacktrace = stacktrace.Callers(pc)
	}
}

// SetLog initialises the Error.Log field with the given message.
// The other Log fields will be empty.
func (e *Error) SetLog(message string) {
	// TODO(axw) pool logs
	e.Log = &model.Log{Message: message}
}

// SetExceptionStacktrace sets the stacktrace for the error,
// skipping the first skip number of frames, excluding the
// SetExceptionStacktrace function.
func (e *Error) SetExceptionStacktrace(skip int) {
	e.Exception.Stacktrace = stacktrace.Stacktrace(skip+1, -1)
}

// SetLogStacktrace sets the stacktrace for the error, skipping
// the first skip number of frames, excluding the SetLogStacktrace
// function.
func (e *Error) SetLogStacktrace(skip int) {
	e.Log.Stacktrace = stacktrace.Stacktrace(skip+1, -1)
}

func errTemporary(err error) bool {
	type temporaryError interface {
		Temporary() bool
	}
	terr, ok := err.(temporaryError)
	return ok && terr.Temporary()
}

func errTimeout(err error) bool {
	type timeoutError interface {
		Timeout() bool
	}
	terr, ok := err.(timeoutError)
	return ok && terr.Timeout()
}
