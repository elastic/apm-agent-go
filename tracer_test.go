package apm_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/internal/apmhostutil"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/transport"
	"go.elastic.co/apm/transport/transporttest"
)

func TestTracerStats(t *testing.T) {
	tracer, err := apm.NewTracer("tracer_testing", "")
	assert.NoError(t, err)
	defer tracer.Close()
	tracer.Transport = transporttest.Discard

	for i := 0; i < 500; i++ {
		tracer.StartTransaction("name", "type").End()
	}
	tracer.Flush(nil)
	assert.Equal(t, apm.TracerStats{
		TransactionsSent: 500,
	}, tracer.Stats())
}

func TestTracerClosedSendNonblocking(t *testing.T) {
	tracer, err := apm.NewTracer("tracer_testing", "")
	assert.NoError(t, err)
	tracer.Close()

	for i := 0; i < 1001; i++ {
		tracer.StartTransaction("name", "type").End()
	}
	assert.Equal(t, uint64(1), tracer.Stats().TransactionsDropped)
}

func TestTracerCloseImmediately(t *testing.T) {
	tracer, err := apm.NewTracer("tracer_testing", "")
	assert.NoError(t, err)
	tracer.Close()
}

func TestTracerFlushEmpty(t *testing.T) {
	tracer, err := apm.NewTracer("tracer_testing", "")
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

	s0 := tx.StartSpan("name", "type", nil)
	s1 := tx.StartSpan("name", "type", nil)
	s2 := tx.StartSpan("name", "type", nil)
	tx.End()

	assert.False(t, s0.Dropped())
	assert.False(t, s1.Dropped())
	assert.True(t, s2.Dropped())
	s0.End()
	s1.End()
	s2.End()

	tracer.Flush(nil)
	assert.Len(t, r.Payloads().Spans, 2)
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
	span := payloads.Spans[0]
	assert.Equal(t, "blam", error0.Exception.Message)
	assert.Equal(t, transaction.ID, error0.TransactionID)
	assert.Equal(t, span.ID, error0.ParentID)
}

func capturePanic(tracer *apm.Tracer, v interface{}) {
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
	_, err := apm.NewTracer("wot!", "")
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

	spans := r.Payloads().Spans
	assert.Len(t, spans, 3)

	// Span 0 took only 9ms, so we don't set its stacktrace.
	assert.Nil(t, spans[0].Stacktrace)

	// Span 1 took the required 10ms, so we set its stacktrace.
	assert.NotNil(t, spans[1].Stacktrace)
	assert.NotEqual(t, spans[1].Stacktrace[0].Function, "TestSpanStackTrace")

	// Span 2 took more than the required 10ms, but its stacktrace
	// was already set; we don't replace it.
	assert.NotNil(t, spans[2].Stacktrace)
	assert.Equal(t, spans[2].Stacktrace[0].Function, "TestSpanStackTrace")
}

func TestTracerRequestSize(t *testing.T) {
	os.Setenv("ELASTIC_APM_API_REQUEST_SIZE", "1KB")
	defer os.Unsetenv("ELASTIC_APM_API_REQUEST_SIZE")

	type times struct {
		start, end time.Time
	}
	requestHandled := make(chan times, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		serverStart := time.Now()
		io.Copy(ioutil.Discard, req.Body)
		serverEnd := time.Now()
		select {
		case requestHandled <- times{start: serverStart, end: serverEnd}:
		default:
		}
	}))
	defer server.Close()

	os.Setenv("ELASTIC_APM_SERVER_URLS", server.URL)
	defer os.Unsetenv("ELASTIC_APM_SERVER_URLS")

	tracer, err := apm.NewTracer("tracer_testing", "")
	require.NoError(t, err)
	defer tracer.Close()
	httpTransport, err := transport.NewHTTPTransport()
	require.NoError(t, err)
	tracer.Transport = httpTransport

	// Send through a bunch of transactions, filling up the API request
	// size, causing the request to be immediately completed.
	clientStart := time.Now()
	for i := 0; i < 500; i++ {
		tracer.StartTransaction("name", "type").End()
		// Yield to the tracer for more predictable timing.
		runtime.Gosched()
	}
	serverTimes := <-requestHandled
	clientEnd := time.Now()
	assert.WithinDuration(t, clientStart, clientEnd, 100*time.Millisecond)
	assert.WithinDuration(t, clientStart, serverTimes.start, 100*time.Millisecond)
	assert.WithinDuration(t, clientEnd, serverTimes.end, 100*time.Millisecond)
}

func TestTracerBufferSize(t *testing.T) {
	os.Setenv("ELASTIC_APM_API_REQUEST_SIZE", "1KB")
	os.Setenv("ELASTIC_APM_API_BUFFER_SIZE", "10KB")
	defer os.Unsetenv("ELASTIC_APM_API_REQUEST_SIZE")
	defer os.Unsetenv("ELASTIC_APM_API_BUFFER_SIZE")

	tracer, recorder := transporttest.NewRecorderTracer()
	defer tracer.Close()
	unblock := make(chan struct{})
	tracer.Transport = blockedTransport{
		Transport: tracer.Transport,
		unblocked: unblock,
	}

	// Send a bunch of transactions, which will be buffered. Because the
	// buffer cannot hold all of them we should expect to see some of the
	// older ones discarded.
	const N = 1000
	for i := 0; i < N; i++ {
		tracer.StartTransaction(fmt.Sprint(i), "type").End()
	}
	close(unblock) // allow requests through now
	for {
		stats := tracer.Stats()
		if stats.TransactionsSent+stats.TransactionsDropped == N {
			require.NotZero(t, stats.TransactionsSent)
			require.NotZero(t, stats.TransactionsDropped)
			break
		}
		tracer.Flush(nil)
	}

	stats := tracer.Stats()
	p := recorder.Payloads()
	assert.Equal(t, int(stats.TransactionsSent), len(p.Transactions))

	// It's possible that the tracer loop receives the flush request after
	// all transactions are in the channel buffer, before any individual
	// transactions make their way through. In most cases we would expect
	// to see the "0" transaction in the request, but that won't be the
	// case if the flush comes first.
	offset := 0
	for i, tx := range p.Transactions {
		if tx.Name != fmt.Sprint(i+offset) {
			require.Equal(t, 0, offset)
			n, err := strconv.Atoi(tx.Name)
			require.NoError(t, err)
			offset = n - i
			t.Logf("found gap of %d after first %d transactions", offset, i)
		}
	}
	assert.NotEqual(t, 0, offset)
}

func TestTracerBodyUnread(t *testing.T) {
	os.Setenv("ELASTIC_APM_API_REQUEST_SIZE", "1KB")
	defer os.Unsetenv("ELASTIC_APM_API_REQUEST_SIZE")

	// Don't consume the request body in the handler; close the connection.
	var requests int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		atomic.AddInt64(&requests, 1)
		w.Header().Set("Connection", "close")
	}))
	defer server.Close()

	os.Setenv("ELASTIC_APM_SERVER_URLS", server.URL)
	defer os.Unsetenv("ELASTIC_APM_SERVER_URLS")

	tracer, err := apm.NewTracer("tracer_testing", "")
	require.NoError(t, err)
	defer tracer.Close()
	httpTransport, err := transport.NewHTTPTransport()
	require.NoError(t, err)
	tracer.Transport = httpTransport

	for atomic.LoadInt64(&requests) <= 1 {
		tracer.StartTransaction("name", "type").End()
	}
	tracer.Flush(nil)
}

func TestTracerMetadata(t *testing.T) {
	tracer, recorder := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.StartTransaction("name", "type").End()
	tracer.Flush(nil)

	// TODO(axw) check other metadata
	system, _, _ := recorder.Metadata()
	container, err := apmhostutil.Container()
	if err != nil {
		assert.Nil(t, system.Container)
	} else {
		require.NotNil(t, system.Container)
		assert.Equal(t, container, system.Container)
	}
}

func TestTracerKubernetesMetadata(t *testing.T) {
	t.Run("no-env", func(t *testing.T) {
		system, _, _ := getSubprocessMetadata(t)
		assert.Nil(t, system.Kubernetes)
	})

	t.Run("namespace-only", func(t *testing.T) {
		system, _, _ := getSubprocessMetadata(t, "KUBERNETES_NAMESPACE=myapp")
		assert.Equal(t, &model.Kubernetes{
			Namespace: "myapp",
		}, system.Kubernetes)
	})

	t.Run("pod-only", func(t *testing.T) {
		system, _, _ := getSubprocessMetadata(t, "KUBERNETES_POD_NAME=luna", "KUBERNETES_POD_UID=oneone!11")
		assert.Equal(t, &model.Kubernetes{
			Pod: &model.KubernetesPod{
				Name: "luna",
				UID:  "oneone!11",
			},
		}, system.Kubernetes)
	})

	t.Run("node-only", func(t *testing.T) {
		system, _, _ := getSubprocessMetadata(t, "KUBERNETES_NODE_NAME=noddy")
		assert.Equal(t, &model.Kubernetes{
			Node: &model.KubernetesNode{
				Name: "noddy",
			},
		}, system.Kubernetes)
	})
}

type blockedTransport struct {
	transport.Transport
	unblocked chan struct{}
}

func (bt blockedTransport) SendStream(ctx context.Context, r io.Reader) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-bt.unblocked:
		return bt.Transport.SendStream(ctx, r)
	}
}
