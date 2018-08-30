package elasticapm_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/transport"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

func TestTracerStats(t *testing.T) {
	tracer, err := elasticapm.NewTracer("tracer_testing", "")
	assert.NoError(t, err)
	defer tracer.Close()
	tracer.Transport = transporttest.Discard

	for i := 0; i < 500; i++ {
		tracer.StartTransaction("name", "type").End()
	}
	tracer.Flush(nil)
	assert.Equal(t, elasticapm.TracerStats{
		TransactionsSent: 500,
	}, tracer.Stats())
}

func TestTracerClosedSendNonblocking(t *testing.T) {
	tracer, err := elasticapm.NewTracer("tracer_testing", "")
	assert.NoError(t, err)
	tracer.Close()

	for i := 0; i < 1001; i++ {
		tracer.StartTransaction("name", "type").End()
	}
	assert.Equal(t, uint64(1), tracer.Stats().TransactionsDropped)
}

func TestTracerCloseImmediately(t *testing.T) {
	tracer, err := elasticapm.NewTracer("tracer_testing", "")
	assert.NoError(t, err)
	tracer.Close()
}

func TestTracerFlushEmpty(t *testing.T) {
	tracer, err := elasticapm.NewTracer("tracer_testing", "")
	assert.NoError(t, err)
	defer tracer.Close()
	tracer.Flush(nil)
}

func TestTracerMaxSpans(t *testing.T) {
	tracer, r := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.SetMaxSpans(2)
	tx := tracer.StartTransaction("name", "type")
	// SetMaxSpans only affects transactions started
	// after the call.
	tracer.SetMaxSpans(99)

	assert.False(t, tx.StartSpan("name", "type", nil).Dropped())
	assert.False(t, tx.StartSpan("name", "type", nil).Dropped())
	assert.True(t, tx.StartSpan("name", "type", nil).Dropped())
	tx.End()

	tracer.Flush(nil)
	payloads := r.Payloads()
	transaction := payloads.Transactions[0]
	assert.Len(t, transaction.Spans, 2)
}

func TestTracerErrors(t *testing.T) {
	tracer, r := transporttest.NewRecorderTracer()
	defer tracer.Close()

	error_ := tracer.NewError(errors.New("zing"))
	error_.Send()
	tracer.Flush(nil)

	payloads := r.Payloads()
	exception := payloads.Errors[0].Exception
	stacktrace := exception.Stacktrace
	assert.Equal(t, "zing", exception.Message)
	assert.Equal(t, "errors", exception.Module)
	assert.Equal(t, "errorString", exception.Type)
	assert.NotEmpty(t, stacktrace)
	assert.Equal(t, "TestTracerErrors", stacktrace[0].Function)
}

func TestTracerErrorFlushes(t *testing.T) {
	tracer, recorder := transporttest.NewRecorderTracer()
	defer tracer.Close()

	payloads := make(chan transporttest.Payloads, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	done := make(chan struct{})
	go func() {
		defer wg.Done()
		for {
			select {
			case <-time.After(10 * time.Millisecond):
				p := recorder.Payloads()
				if len(p.Errors)+len(p.Transactions) > 0 {
					payloads <- p
					return
				}
			case <-done:
			}
		}
	}()
	defer wg.Wait()
	defer close(done)

	// Sending a transaction should not cause a request
	// to be sent immediately.
	tracer.StartTransaction("name", "type").End()
	select {
	case <-time.After(200 * time.Millisecond):
	case p := <-payloads:
		t.Fatalf("unexpected payloads: %+v", p)
	}

	// Sending an error flushes the request body.
	tracer.NewError(errors.New("zing")).Send()
	select {
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for request")
	case p := <-payloads:
		assert.Len(t, p.Transactions, 1)
		assert.Len(t, p.Errors, 1)
	}

	// TODO(axw) tracer.Close should wait for the current request
	// to complete, at least for a short amount of time.
	tracer.Flush(nil)
}

func TestTracerRecovered(t *testing.T) {
	tracer, r := transporttest.NewRecorderTracer()
	defer tracer.Close()

	capturePanic(tracer, "blam")
	tracer.Flush(nil)

	payloads := r.Payloads()
	error0 := payloads.Errors[0]
	transaction := payloads.Transactions[0]
	span := transaction.Spans[0]
	assert.Equal(t, "blam", error0.Exception.Message)
	assert.Equal(t, transaction.ID, error0.TransactionID)
	assert.Equal(t, span.ID, error0.ParentID)
}

func capturePanic(tracer *elasticapm.Tracer, v interface{}) {
	tx := tracer.StartTransaction("name", "type")
	defer tx.End()
	span := tx.StartSpan("name", "type", nil)
	defer span.End()
	defer func() {
		if v := recover(); v != nil {
			e := tracer.Recovered(v)
			e.SetSpan(span)
			e.Send()
		}
	}()
	panic(v)
}

func TestTracerServiceNameValidation(t *testing.T) {
	_, err := elasticapm.NewTracer("wot!", "")
	assert.EqualError(t, err, `invalid service name "wot!": character '!' is not in the allowed set (a-zA-Z0-9 _-)`)
}

func TestSpanStackTrace(t *testing.T) {
	tracer, r := transporttest.NewRecorderTracer()
	defer tracer.Close()
	tracer.SetSpanFramesMinDuration(10 * time.Millisecond)

	tx := tracer.StartTransaction("name", "type")
	s := tx.StartSpan("name", "type", nil)
	s.Duration = 9 * time.Millisecond
	s.End()
	s = tx.StartSpan("name", "type", nil)
	s.Duration = 10 * time.Millisecond
	s.End()
	s = tx.StartSpan("name", "type", nil)
	s.SetStacktrace(1)
	s.Duration = 11 * time.Millisecond
	s.End()
	tx.End()
	tracer.Flush(nil)

	transaction := r.Payloads().Transactions[0]
	assert.Len(t, transaction.Spans, 3)

	// Span 0 took only 9ms, so we don't set its stacktrace.
	span0 := transaction.Spans[0]
	assert.Nil(t, span0.Stacktrace)

	// Span 1 took the required 10ms, so we set its stacktrace.
	span1 := transaction.Spans[1]
	assert.NotNil(t, span1.Stacktrace)
	assert.NotEqual(t, span1.Stacktrace[0].Function, "TestSpanStackTrace")

	// Span 2 took more than the required 10ms, but its stacktrace
	// was already set; we don't replace it.
	span2 := transaction.Spans[2]
	assert.NotNil(t, span2.Stacktrace)
	assert.Equal(t, span2.Stacktrace[0].Function, "TestSpanStackTrace")
}

func TestTracerRequestSize(t *testing.T) {
	os.Setenv("ELASTIC_APM_API_REQUEST_SIZE", "1024")
	defer os.Unsetenv("ELASTIC_APM_API_REQUEST_SIZE")

	requestHandled := make(chan struct{}, 1)
	var serverStart, serverEnd time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		serverStart = time.Now()
		io.Copy(ioutil.Discard, req.Body)
		serverEnd = time.Now()
		select {
		case requestHandled <- struct{}{}:
		case <-req.Context().Done():
		}
	}))
	defer server.Close()

	tracer, err := elasticapm.NewTracer("tracer_testing", "")
	require.NoError(t, err)
	defer tracer.Close()
	httpTransport, err := transport.NewHTTPTransport(server.URL, "")
	require.NoError(t, err)
	tracer.Transport = httpTransport

	// Send through a bunch of requests, filling up the API request
	// size, causing the request to be immediately completed.
	clientStart := time.Now()
	for i := 0; i < 1000; i++ {
		tracer.StartTransaction("name", "type").End()
	}
	<-requestHandled
	clientEnd := time.Now()
	assert.WithinDuration(t, clientStart, clientEnd, 100*time.Millisecond)
	assert.WithinDuration(t, clientStart, serverStart, 100*time.Millisecond)
	assert.WithinDuration(t, clientEnd, serverEnd, 100*time.Millisecond)
}

func TestTracerBufferSize(t *testing.T) {
	os.Setenv("ELASTIC_APM_API_REQUEST_SIZE", "1024")
	os.Setenv("ELASTIC_APM_API_BUFFER_SIZE", "10240")
	defer os.Unsetenv("ELASTIC_APM_API_REQUEST_SIZE")
	defer os.Unsetenv("ELASTIC_APM_API_BUFFER_SIZE")

	tracer, recorder := transporttest.NewRecorderTracer()
	defer tracer.Close()
	unblocks := make(chan chan struct{})
	tracer.Transport = blockedTransport{
		Transport: tracer.Transport,
		C:         unblocks,
	}
	for i := 0; i < 1000; i++ {
		tracer.StartTransaction(fmt.Sprint(i), "type").End()
	}
	// Don't allow anything through until all transactions are
	// buffered. This will cause some of the transactions to be
	// dropped in favour of new ones.
	(<-unblocks) <- struct{}{}
	tracer.Flush(nil)

	// The first request should have all its transactions in order,
	// since they get encoded into the request buffer immediately.
	p := recorder.Payloads()
	assert.NotEmpty(t, p.Transactions)
	for i, tx := range p.Transactions {
		assert.Equal(t, fmt.Sprint(i), tx.Name)
	}

	// Let through the next request, which will have been filled
	// from the buffer _after_ dropping some objects. There should
	// be a gap in the transaction names.
	(<-unblocks) <- struct{}{}
	tracer.Flush(nil)
	p2 := recorder.Payloads()
	assert.NotEqual(t, len(p.Transactions), len(p2.Transactions))
	assert.NotEqual(t, fmt.Sprint(len(p.Transactions)), p2.Transactions[len(p.Transactions)].Name)

	// We record the type of event in the buffer, in order to keep the dropped stats accurate.
	assert.Equal(t, uint64(1000-len(p2.Transactions)), tracer.Stats().TransactionsDropped)
}

func TestTracerBodyUnread(t *testing.T) {
	os.Setenv("ELASTIC_APM_API_REQUEST_SIZE", "1024")
	defer os.Unsetenv("ELASTIC_APM_API_REQUEST_SIZE")

	// Don't consume the request body in the handler; close the connection.
	var requests int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		atomic.AddInt64(&requests, 1)
		w.Header().Set("Connection", "close")
	}))
	defer server.Close()

	tracer, err := elasticapm.NewTracer("tracer_testing", "")
	require.NoError(t, err)
	defer tracer.Close()
	httpTransport, err := transport.NewHTTPTransport(server.URL, "")
	require.NoError(t, err)
	tracer.Transport = httpTransport

	for atomic.LoadInt64(&requests) <= 1 {
		tracer.StartTransaction("name", "type").End()
	}
	tracer.Flush(nil)
}

type blockedTransport struct {
	transport.Transport
	C chan chan struct{}
}

func (bt blockedTransport) SendStream(ctx context.Context, r io.Reader) error {
	ch := make(chan struct{})
	select {
	case <-ctx.Done():
		return ctx.Err()
	case bt.C <- ch:
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ch:
		}
		return bt.Transport.SendStream(ctx, r)
	}
}
