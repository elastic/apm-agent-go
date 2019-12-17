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
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
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
	statusCode, _ := mustGET(ctx, requestURL.String())
	assert.Equal(t, http.StatusTeapot, statusCode)
	tx.End()
	tracer.Flush(nil)

	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, 1)
	require.Len(t, payloads.Spans, 1)
	transaction := payloads.Transactions[0]
	span := payloads.Spans[0]

	serverAddr := server.Listener.Addr().(*net.TCPAddr)

	assert.Equal(t, transaction.ID, span.ParentID)
	assert.Equal(t, "GET "+serverAddr.String(), span.Name)
	assert.Equal(t, "external", span.Type)
	assert.Equal(t, "http", span.Subtype)
	assert.Equal(t, &model.SpanContext{
		Destination: &model.DestinationSpanContext{
			Address: serverAddr.IP.String(),
			Port:    serverAddr.Port,
			Service: &model.DestinationServiceSpanContext{
				Type:     "external",
				Name:     "http://" + serverAddr.String(),
				Resource: serverAddr.String(),
			},
		},
		HTTP: &model.HTTPSpanContext{
			// Note no user info included in server.URL.
			URL:        serverURL,
			StatusCode: statusCode,
		},
	}, span.Context)
}

func TestClientTraceContextHeaders(t *testing.T) {
	t.Run("with-elastic-apm-traceparent", func(t *testing.T) {
		testClientTraceContextHeaders(t, "Elastic-Apm-Traceparent", "Traceparent")
	})
	t.Run("without-elastic-apm-traceparent", func(t *testing.T) {
		os.Setenv("ELASTIC_APM_USE_ELASTIC_TRACEPARENT_HEADER", "true")
		defer os.Unsetenv("ELASTIC_APM_USE_ELASTIC_TRACEPARENT_HEADER")
		testClientTraceContextHeaders(t, "Traceparent")
	})
}

func testClientTraceContextHeaders(t *testing.T, traceparentHeaders ...string) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		headers := make(map[string]string)
		for k, vs := range req.Header {
			headers[k] = strings.Join(vs, " ")
		}
		json.NewEncoder(w).Encode(headers)
	}))
	defer server.Close()

	tx := tracer.StartTransactionOptions("name", "type", apm.TransactionOptions{
		TraceContext: apm.TraceContext{
			Trace:   apm.TraceID{1},
			Span:    apm.SpanID{1},
			Options: apm.TraceOptions(0).WithRecorded(true),
			State:   apm.NewTraceState(apm.TraceStateEntry{Key: "vendor", Value: "tracestate"}),
		},
	})
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	_, responseBody := mustGET(ctx, server.URL)
	tx.End()
	tracer.Flush(nil)

	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, 1)
	require.Len(t, payloads.Spans, 1)
	span := payloads.Spans[0]

	headers := make(map[string]string)
	err := json.Unmarshal([]byte(responseBody), &headers)
	require.NoError(t, err)

	traceparentValue := apmhttp.FormatTraceparentHeader(apm.TraceContext{
		Trace:   apm.TraceID(span.TraceID),
		Span:    apm.SpanID(span.ID),
		Options: apm.TraceOptions(0).WithRecorded(true),
	})
	for _, header := range traceparentHeaders {
		require.Contains(t, headers, header)
		assert.Equal(t, traceparentValue, headers[header])
	}

	require.Contains(t, headers, "Tracestate")
	assert.Equal(t, "vendor=tracestate", headers["Tracestate"])
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("hello"))
		w.(http.Flusher).Flush()
		time.Sleep(500 * time.Millisecond)
		w.Write([]byte("world"))
	}))
	defer server.Close()

	var elapsed time.Duration
	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		before := time.Now()
		// mustGET reads the body, so it should not return until the handler completes.
		mustGET(ctx, server.URL)
		elapsed = time.Since(before)
	})

	require.Len(t, spans, 1)
	assert.InEpsilon(t,
		elapsed,
		spans[0].Duration*float64(time.Millisecond),
		0.1, // 10% error
	)
}

func TestClientCancelRequest(t *testing.T) {
	done := make(chan struct{})
	canceled := make(chan struct{}, 1)
	transport := &cancelRequester{
		RoundTripper: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			// We don't wait on req.Context().Done(), because there's no
			// guarantee that CancelRequest is called after RoundTrip returns.
			<-done
			return nil, errors.New("nope")
		}),
		cancelRequest: func(*http.Request) {
			select {
			case canceled <- struct{}{}:
				close(done)
			default:
			}
		},
	}
	apmtest.WithTransaction(func(ctx context.Context) {
		client := &http.Client{
			Transport: apmhttp.WrapRoundTripper(transport),
			Timeout:   time.Nanosecond,
		}
		_, err := ctxhttp.Get(ctx, client, "http://testing.invalid")
		require.Error(t, err)
	})

	select {
	case <-canceled:
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out waiting for CancelRequest to be called")
	}
}

func TestWithClientRequestName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer server.Close()

	option := apmhttp.WithClientRequestName(func(_ *http.Request) string {
		return "http://test"
	})

	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		mustGET(ctx, server.URL, option)
	})

	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, "http://test", span.Name)
}

func mustGET(ctx context.Context, url string, o ...apmhttp.ClientOption) (statusCode int, responseBody string) {
	client := apmhttp.WrapClient(http.DefaultClient, o...)
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

type cancelRequester struct {
	http.RoundTripper
	cancelRequest func(*http.Request)
}

func (r *cancelRequester) CancelRequest(req *http.Request) {
	r.cancelRequest(req)
}

type RoundTripFunc func(*http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
