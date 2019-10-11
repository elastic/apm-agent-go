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
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/internal/apmhostutil"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/transport"
	"go.elastic.co/apm/transport/transporttest"
)

func TestTracerStats(t *testing.T) {
	tracer := apmtest.NewDiscardTracer()
	defer tracer.Close()

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
	test := func(n int) {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			tracer, r := transporttest.NewRecorderTracer()
			defer tracer.Close()

			tracer.SetMaxSpans(n)
			tx := tracer.StartTransaction("name", "type")
			defer tx.End()

			// SetMaxSpans only affects transactions started
			// after the call.
			tracer.SetMaxSpans(99)

			for i := 0; i < n; i++ {
				span := tx.StartSpan("name", "type", nil)
				assert.False(t, span.Dropped())
				span.End()
			}
			span := tx.StartSpan("name", "type", nil)
			assert.True(t, span.Dropped())
			span.End()

			tracer.Flush(nil)
			assert.Len(t, r.Payloads().Spans, n)
		})
	}
	test(0)
	test(23)
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
	require.NotEmpty(t, stacktrace)
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
		var last int
		for {
			select {
			case <-time.After(10 * time.Millisecond):
				p := recorder.Payloads()
				if n := len(p.Errors) + len(p.Transactions); n > last {
					last = n
					payloads <- p
				}
			case <-done:
				return
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
	deadline := time.After(2 * time.Second)
	for {
		var p transporttest.Payloads
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for request")
		case p = <-payloads:
		}
		if len(p.Errors) != 0 {
			assert.Len(t, p.Errors, 1)
			break
		}
		// The transport may not have decoded
		// the error yet, continue waiting.
	}
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
	require.Len(t, spans, 3)

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

	// Set the request time to some very long duration,
	// to highlight the fact that the request size is
	// the cause of request completion.
	os.Setenv("ELASTIC_APM_API_REQUEST_TIME", "60s")
	defer os.Unsetenv("ELASTIC_APM_API_REQUEST_TIME")

	requestHandled := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		io.Copy(ioutil.Discard, req.Body)
		requestHandled <- struct{}{}
	}))
	defer server.Close()

	os.Setenv("ELASTIC_APM_SERVER_URLS", server.URL)
	defer os.Unsetenv("ELASTIC_APM_SERVER_URLS")

	httpTransport, err := transport.NewHTTPTransport()
	require.NoError(t, err)
	tracer, err := apm.NewTracerOptions(apm.TracerOptions{
		ServiceName: "tracer_testing",
		Transport:   httpTransport,
	})
	require.NoError(t, err)
	defer tracer.Close()

	// Send through a bunch of transactions, filling up the API request
	// size, causing the request to be immediately completed.
	clientStart := time.Now()
	for i := 0; i < 500; i++ {
		tracer.StartTransaction("name", "type").End()
		// Yield to the tracer for more predictable timing.
		runtime.Gosched()
	}
	<-requestHandled
	clientEnd := time.Now()
	assert.Condition(t, func() bool {
		// Should be considerably less than 10s, which is
		// considerably less than the configured 60s limit.
		return clientEnd.Sub(clientStart) < 10*time.Second
	})
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

	httpTransport, err := transport.NewHTTPTransport()
	require.NoError(t, err)
	tracer, err := apm.NewTracerOptions(apm.TracerOptions{
		ServiceName: "tracer_testing",
		Transport:   httpTransport,
	})
	require.NoError(t, err)
	defer tracer.Close()

	for atomic.LoadInt64(&requests) <= 1 {
		tracer.StartTransaction("name", "type").End()
	}
}

func TestTracerMetadata(t *testing.T) {
	tracer, recorder := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.StartTransaction("name", "type").End()
	tracer.Flush(nil)

	// TODO(axw) check other metadata
	system, _, _, _ := recorder.Metadata()
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
		system, _, _, _ := getSubprocessMetadata(t)
		assert.Nil(t, system.Kubernetes)
	})

	t.Run("namespace-only", func(t *testing.T) {
		system, _, _, _ := getSubprocessMetadata(t, "KUBERNETES_NAMESPACE=myapp")
		assert.Equal(t, &model.Kubernetes{
			Namespace: "myapp",
		}, system.Kubernetes)
	})

	t.Run("pod-only", func(t *testing.T) {
		system, _, _, _ := getSubprocessMetadata(t, "KUBERNETES_POD_NAME=luna", "KUBERNETES_POD_UID=oneone!11")
		assert.Equal(t, &model.Kubernetes{
			Pod: &model.KubernetesPod{
				Name: "luna",
				UID:  "oneone!11",
			},
		}, system.Kubernetes)
	})

	t.Run("node-only", func(t *testing.T) {
		system, _, _, _ := getSubprocessMetadata(t, "KUBERNETES_NODE_NAME=noddy")
		assert.Equal(t, &model.Kubernetes{
			Node: &model.KubernetesNode{
				Name: "noddy",
			},
		}, system.Kubernetes)
	})
}

func TestTracerActive(t *testing.T) {
	tracer, _ := transporttest.NewRecorderTracer()
	defer tracer.Close()
	assert.True(t, tracer.Active())

	// Kick off calls to tracer.Active concurrently
	// with the tracer.Close, to test that we ensure
	// there are no data races.
	go func() {
		for i := 0; i < 100; i++ {
			tracer.Active()
		}
	}()
}

func TestTracerCaptureHeaders(t *testing.T) {
	tracer, recorder := transporttest.NewRecorderTracer()
	defer tracer.Close()

	req, err := http.NewRequest("GET", "http://testing.invalid", nil)
	require.NoError(t, err)
	req.Header.Set("foo", "bar")
	respHeaders := make(http.Header)
	respHeaders.Set("baz", "qux")

	for _, enabled := range []bool{false, true} {
		tracer.SetCaptureHeaders(enabled)
		tx := tracer.StartTransaction("name", "type")
		tx.Context.SetHTTPRequest(req)
		tx.Context.SetHTTPResponseHeaders(respHeaders)
		tx.Context.SetHTTPStatusCode(202)
		tx.End()
	}

	tracer.Flush(nil)
	payloads := recorder.Payloads()
	require.Len(t, payloads.Transactions, 2)

	for i, enabled := range []bool{false, true} {
		tx := payloads.Transactions[i]
		require.NotNil(t, tx.Context.Request)
		require.NotNil(t, tx.Context.Response)
		if enabled {
			assert.NotNil(t, tx.Context.Request.Headers)
			assert.NotNil(t, tx.Context.Response.Headers)
		} else {
			assert.Nil(t, tx.Context.Request.Headers)
			assert.Nil(t, tx.Context.Response.Headers)
		}
	}
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
