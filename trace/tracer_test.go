package trace_test

import (
	"log"
	"runtime"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/apm-agent-go/trace"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

func TestTracerStats(t *testing.T) {
	tracer, err := trace.NewTracer("tracer.testing", "")
	if err != nil {
		log.Fatal(err)
	}
	defer tracer.Close()
	tracer.Transport = transporttest.Discard

	for i := 0; i < 500; i++ {
		tracer.StartTransaction("name", "type").Done(-1)
	}
	tracer.Flush(nil)
	assert.Equal(t, trace.TracerStats{
		TransactionsSent: 500,
	}, tracer.Stats())
}

func TestTracerFlushInterval(t *testing.T) {
	tracer, err := trace.NewTracer("tracer.testing", "")
	if err != nil {
		log.Fatal(err)
	}
	defer tracer.Close()
	tracer.Transport = transporttest.Discard

	interval := time.Second
	tracer.SetFlushInterval(interval)

	before := time.Now()
	tracer.StartTransaction("name", "type").Done(-1)
	assert.Equal(t, trace.TracerStats{TransactionsSent: 0}, tracer.Stats())
	for tracer.Stats().TransactionsSent == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	assert.WithinDuration(t, before.Add(interval), time.Now(), 100*time.Millisecond)
}

func TestTracerMaxQueueSize(t *testing.T) {
	tracer, err := trace.NewTracer("tracer.testing", "")
	if err != nil {
		log.Fatal(err)
	}
	defer tracer.Close()

	// Prevent any transactions from being sent.
	tracer.Transport = transporttest.ErrorTransport{errors.New("nope")}

	// Enqueue 10 transactions with a queue size of 5;
	// we should see 5 transactons dropped.
	tracer.SetMaxQueueSize(5)
	for i := 0; i < 10; i++ {
		tracer.StartTransaction("name", "type").Done(-1)
	}
	for tracer.Stats().TransactionsDropped < 5 {
		time.Sleep(10 * time.Millisecond)
	}
	assert.Equal(t, trace.TracerStats{
		Errors: trace.TracerStatsErrors{
			SendTransactions: 1,
		},
		TransactionsDropped: 5,
	}, tracer.Stats())
}

func TestTracerRetryTimer(t *testing.T) {
	tracer, err := trace.NewTracer("tracer.testing", "")
	if err != nil {
		log.Fatal(err)
	}
	defer tracer.Close()

	// Prevent any transactions from being sent.
	tracer.Transport = transporttest.ErrorTransport{errors.New("nope")}

	interval := time.Second
	tracer.SetFlushInterval(interval)
	tracer.SetMaxQueueSize(1)

	before := time.Now()
	tracer.StartTransaction("name", "type").Done(-1)
	for tracer.Stats().Errors.SendTransactions < 1 {
		time.Sleep(10 * time.Millisecond)
	}
	assert.Equal(t, trace.TracerStats{
		Errors: trace.TracerStatsErrors{
			SendTransactions: 1,
		},
	}, tracer.Stats())

	// Send another transaction, which should cause the
	// existing transaction to be dropped, but should not
	// preempt the retry timer.
	tracer.StartTransaction("name", "type").Done(-1)
	for tracer.Stats().Errors.SendTransactions < 2 {
		time.Sleep(10 * time.Millisecond)
	}
	assert.WithinDuration(t, before.Add(interval), time.Now(), 100*time.Millisecond)
	assert.Equal(t, trace.TracerStats{
		Errors: trace.TracerStatsErrors{
			SendTransactions: 2,
		},
		TransactionsDropped: 1,
	}, tracer.Stats())
}

func TestTracerMaxSpans(t *testing.T) {
	var r transporttest.RecorderTransport
	tracer, err := trace.NewTracer("tracer.testing", "")
	if err != nil {
		log.Fatal(err)
	}
	defer tracer.Close()
	tracer.Transport = &r

	tracer.SetMaxSpans(2)
	tx := tracer.StartTransaction("name", "type")
	// SetMaxSpans only affects transactions started
	// after the call.
	tracer.SetMaxSpans(99)

	s0 := tx.StartSpan("name", "type", nil)
	s1 := tx.StartSpan("name", "type", nil)
	s2 := tx.StartSpan("name", "type", nil)
	tx.Done(-1)

	assert.False(t, s0.Dropped())
	assert.False(t, s1.Dropped())
	assert.True(t, s2.Dropped())

	tracer.Flush(nil)
	payloads := r.Payloads()
	assert.Len(t, payloads, 1)
	transactions := payloads[0]["transactions"].([]interface{})
	assert.Len(t, transactions, 1)
	transaction := transactions[0].(map[string]interface{})
	assert.Len(t, transaction["spans"], 2)
}

func TestTracerErrors(t *testing.T) {
	var r transporttest.RecorderTransport
	tracer, err := trace.NewTracer("tracer.testing", "")
	if err != nil {
		log.Fatal(err)
	}
	defer tracer.Close()
	tracer.Transport = &r

	error_ := tracer.NewError()
	error_.SetException(&testError{
		"zing", newErrorsStackTrace(0, 2),
	})
	error_.Send()
	tracer.Flush(nil)

	payloads := r.Payloads()
	assert.Len(t, payloads, 1)
	errors := payloads[0]["errors"].([]interface{})
	assert.Len(t, errors, 1)
	exception := errors[0].(map[string]interface{})["exception"].(map[string]interface{})
	assert.Equal(t, "zing", exception["message"])
	assert.Equal(t, "github.com/elastic/apm-agent-go/trace_test", exception["module"])
	assert.Equal(t, "testError", exception["type"])
	stacktrace := exception["stacktrace"].([]interface{})
	assert.Len(t, stacktrace, 2)
	frame0 := stacktrace[0].(map[string]interface{})
	frame1 := stacktrace[1].(map[string]interface{})
	assert.Equal(t, "newErrorsStackTrace", frame0["function"])
	assert.Equal(t, "TestTracerErrors", frame1["function"])
}

func TestTracerErrorsBuffered(t *testing.T) {
	// TODO(axw) show that errors are buffered,
	// dropped when full, and sent when possible.
}

type testLogger struct {
	t *testing.T
}

func (l testLogger) Debugf(format string, args ...interface{}) {
	l.t.Logf("[DEBUG] "+format, args...)
}

func (l testLogger) Errorf(format string, args ...interface{}) {
	l.t.Logf("[ERROR] "+format, args...)
}

type testError struct {
	message    string
	stackTrace errors.StackTrace
}

func (e *testError) Error() string {
	return e.message
}

func (e *testError) StackTrace() errors.StackTrace {
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
