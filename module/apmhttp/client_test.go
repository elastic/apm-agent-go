package apmhttp_test

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context/ctxhttp"

	"go.elastic.co/apm"
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
