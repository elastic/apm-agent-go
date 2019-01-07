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

package apmgokit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/transport/transporttest"
)

func Example_httpServer() {
	// Create your go-kit/kit/transport/http.Server as usual, without any tracing middleware.
	endpoint := func(ctx context.Context, req interface{}) (interface{}, error) {
		// The middleware added to the underlying gRPC server will be propagate
		// a transaction to the context passed to your endpoint. You can then
		// report endpoint-specific spans using apm.StartSpan.
		span, ctx := apm.StartSpan(ctx, "name", "endpoint")
		defer span.End()
		return nil, nil
	}
	server := kithttp.NewServer(
		endpoint,
		kithttp.NopRequestDecoder,
		func(_ context.Context, w http.ResponseWriter, _ interface{}) error { return nil },
	)

	// Use apmhttp.Wrap (from module/apmhttp) to instrument the
	// kit/transport/http.Server. This will trace all incoming requests.
	http.ListenAndServe("localhost:1234", apmhttp.Wrap(server))
}

func Example_httpClient() {
	// When constructing the kit/transport/http.Client, pass in an http.Client
	// instrumented using apmhttp.WrapClient (from module/apmhttp). This will
	// trace all outgoing requests, as long as the context supplied to methods
	// include an apm.Transaction.
	client := kithttp.NewClient(
		"GET", &url.URL{ /*...*/ },
		kithttp.EncodeJSONRequest,
		func(_ context.Context, r *http.Response) (interface{}, error) { return nil, nil },
		kithttp.SetClient(apmhttp.WrapClient(http.DefaultClient)),
	)

	tx := apm.DefaultTracer.StartTransaction("name", "type")
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	defer tx.End()

	_, err := client.Endpoint()(ctx, struct{}{})
	if err != nil {
		panic(err)
	}
}

func TestHTTPTransport(t *testing.T) {
	serverTracer, serverRecorder := transporttest.NewRecorderTracer()
	defer serverTracer.Close()

	endpoint := func(ctx context.Context, request interface{}) (response interface{}, err error) {
		span, ctx := apm.StartSpan(ctx, "name", "type")
		defer span.End()
		return struct{}{}, nil
	}

	server := httptest.NewServer(apmhttp.Wrap(kithttp.NewServer(
		endpoint,
		kithttp.NopRequestDecoder,
		func(_ context.Context, w http.ResponseWriter, _ interface{}) error {
			w.Header().Set("foo", "bar")
			w.WriteHeader(http.StatusTeapot)
			return nil
		},
	), apmhttp.WithTracer(serverTracer)))
	defer server.Close()

	url, err := url.Parse(server.URL)
	require.NoError(t, err)
	url.Path = "/foo"
	client := kithttp.NewClient(
		"GET", url,
		kithttp.EncodeJSONRequest,
		func(_ context.Context, r *http.Response) (interface{}, error) {
			assert.Equal(t, http.StatusTeapot, r.StatusCode)
			return nil, nil
		},
		kithttp.SetClient(apmhttp.WrapClient(http.DefaultClient)),
	)
	clientTransaction, clientSpans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		client.Endpoint()(ctx, struct{}{})
	})
	require.Len(t, clientSpans, 1)

	serverTracer.Flush(nil)
	payloads := serverRecorder.Payloads()
	require.Len(t, payloads.Transactions, 1)
	require.Len(t, payloads.Spans, 1)

	assert.Equal(t, "GET /foo", payloads.Transactions[0].Name)
	assert.Equal(t, "name", payloads.Spans[0].Name)
	assert.Equal(t, payloads.Transactions[0].ID, payloads.Spans[0].ParentID)
	assert.Equal(t, clientTransaction.TraceID, payloads.Transactions[0].TraceID)
	assert.Equal(t, clientSpans[0].ID, payloads.Transactions[0].ParentID)

	responseContext := payloads.Transactions[0].Context.Response
	assert.Equal(t, http.StatusTeapot, responseContext.StatusCode)
	assert.Equal(t, model.Headers{
		{Key: "Foo", Values: []string{"bar"}},
	}, responseContext.Headers)
}
