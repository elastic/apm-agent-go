package apmhttp_test

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context/ctxhttp"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/transport/transporttest"
)

func TestClient(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	mux := http.NewServeMux()
	mux.Handle("/foo", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte(req.Header.Get("Elastic-Apm-Traceparent")))
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	serverURL.Path = "/foo"

	// Add user info to the URL; it should be stripped off.
	requestURL := *serverURL
	requestURL.User = url.UserPassword("root", "hunter2")

	tx := tracer.StartTransaction("name", "type")
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	client := apmhttp.WrapClient(http.DefaultClient)
	resp, err := ctxhttp.Get(ctx, client, requestURL.String())
	assert.NoError(t, err)
	responseBody, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusTeapot, resp.StatusCode)
	tx.End()
	tracer.Flush(nil)

	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, 1)
	require.Len(t, payloads.Spans, 1)
	transaction := payloads.Transactions[0]
	span := payloads.Spans[0]

	assert.Equal(t, "GET "+server.Listener.Addr().String(), span.Name)
	assert.Equal(t, "ext.http", span.Type)
	assert.Equal(t, &model.SpanContext{
		HTTP: &model.HTTPSpanContext{
			// Note no user info included in server.URL.
			URL: serverURL,
		},
	}, span.Context)

	clientTraceContext, err := apmhttp.ParseTraceparentHeader(string(responseBody))
	assert.NoError(t, err)
	assert.Equal(t, span.TraceID, model.TraceID(clientTraceContext.Trace))
	assert.Equal(t, span.ID, model.SpanID(clientTraceContext.Span))
	assert.Equal(t, transaction.ID, span.ParentID)
}

func TestClientSpanDropped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(req.Header.Get("Elastic-Apm-Traceparent")))
	}))
	defer server.Close()

	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.SetMaxSpans(1)
	tx := tracer.StartTransaction("name", "type")
	ctx := apm.ContextWithTransaction(context.Background(), tx)

	var responseBodies []string
	for i := 0; i < 2; i++ {
		client := apmhttp.WrapClient(http.DefaultClient)
		resp, err := ctxhttp.Get(ctx, client, server.URL)
		assert.NoError(t, err)
		responseBody, err := ioutil.ReadAll(resp.Body)
		if !assert.NoError(t, err) {
			resp.Body.Close()
			return
		}
		responseBodies = append(responseBodies, string(responseBody))
	}

	tx.End()
	tracer.Flush(nil)
	payloads := transport.Payloads()
	require.Len(t, payloads.Spans, 1)
	transaction := payloads.Transactions[0]
	span := payloads.Spans[0] // for first request

	clientTraceContext, err := apmhttp.ParseTraceparentHeader(string(responseBodies[0]))
	require.NoError(t, err)
	assert.Equal(t, span.TraceID, model.TraceID(clientTraceContext.Trace))
	assert.Equal(t, span.ID, model.SpanID(clientTraceContext.Span))

	clientTraceContext, err = apmhttp.ParseTraceparentHeader(string(responseBodies[1]))
	require.NoError(t, err)
	assert.Equal(t, transaction.TraceID, model.TraceID(clientTraceContext.Trace))
	assert.Equal(t, transaction.ID, model.SpanID(clientTraceContext.Span))
}

func TestClientError(t *testing.T) {
	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		client := apmhttp.WrapClient(http.DefaultClient)
		resp, err := ctxhttp.Get(ctx, client, "http://testing.invalid")
		if !assert.Error(t, err) {
			resp.Body.Close()
		}
	})
	require.Len(t, spans, 1)
}

func TestClientDuration(t *testing.T) {
	const delay = 500 * time.Millisecond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("hello"))
		w.(http.Flusher).Flush()
		time.Sleep(delay)
		w.Write([]byte("world"))
	}))
	defer server.Close()

	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		client := apmhttp.WrapClient(http.DefaultClient)

		resp, err := ctxhttp.Get(ctx, client, server.URL)
		assert.NoError(t, err)
		defer resp.Body.Close()
		io.Copy(ioutil.Discard, resp.Body)
	})

	require.Len(t, spans, 1)
	span := spans[0]

	assert.Equal(t, "GET "+server.Listener.Addr().String(), span.Name)
	assert.Equal(t, "ext.http", span.Type)
	assert.InDelta(t, delay/time.Millisecond, span.Duration, 100)
}
