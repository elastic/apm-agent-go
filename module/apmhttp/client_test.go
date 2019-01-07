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

package apmhttp_test

import (
	"context"
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
	statusCode, responseBody := mustGET(ctx, requestURL.String())
	assert.Equal(t, http.StatusTeapot, statusCode)
	tx.End()
	tracer.Flush(nil)

	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, 1)
	require.Len(t, payloads.Spans, 1)
	transaction := payloads.Transactions[0]
	span := payloads.Spans[0]

	assert.Equal(t, "GET "+server.Listener.Addr().String(), span.Name)
	assert.Equal(t, "external", span.Type)
	assert.Equal(t, "http", span.Subtype)
	assert.Equal(t, &model.SpanContext{
		HTTP: &model.HTTPSpanContext{
			// Note no user info included in server.URL.
			URL:        serverURL,
			StatusCode: statusCode,
		},
	}, span.Context)

	clientTraceContext, err := apmhttp.ParseTraceparentHeader(responseBody)
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
		_, responseBody := mustGET(ctx, server.URL)
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

func TestClientTransactionUnsampled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(req.Header.Get("Elastic-Apm-Traceparent")))
	}))
	defer server.Close()

	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()
	tracer.SetSampler(apm.NewRatioSampler(0)) // sample nothing

	tx := tracer.StartTransaction("name", "type")
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	_, responseBody := mustGET(ctx, server.URL)
	tx.End()
	tracer.Flush(nil)

	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, 1)
	require.Len(t, payloads.Spans, 0)
	transaction := payloads.Transactions[0]

	clientTraceContext, err := apmhttp.ParseTraceparentHeader(string(responseBody))
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
		// mustGET reads the body, so it should not return until the handler completes.
		mustGET(ctx, server.URL)
	})

	require.Len(t, spans, 1)
	span := spans[0]

	assert.Equal(t, "GET "+server.Listener.Addr().String(), span.Name)
	assert.InDelta(t, delay/time.Millisecond, span.Duration, 100)
}

func mustGET(ctx context.Context, url string) (statusCode int, responseBody string) {
	client := apmhttp.WrapClient(http.DefaultClient)
	resp, err := ctxhttp.Get(ctx, client, url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return resp.StatusCode, string(body)
}
