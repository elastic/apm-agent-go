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

package apmfasthttp_test

import (
	"bufio"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"go.elastic.co/apm/module/apmfasthttp/v2"
	"go.elastic.co/apm/v2/model"
	"go.elastic.co/apm/v2/transport/transporttest"
)

type assertFunc func(model.Transaction, *http.Response)

func testServer(t *testing.T, s *fasthttp.Server, wg *sync.WaitGroup, assertFn assertFunc) {
	t.Helper()

	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	shutdown := make(chan error)
	defer close(shutdown)

	ln, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)

	s.Handler = apmfasthttp.Wrap(s.Handler, apmfasthttp.WithTracer(tracer))

	go func() {
		shutdown <- s.Serve(ln)
	}()

	addr := ln.Addr().String()

	resp, err := http.Get("http://" + addr)
	require.NoError(t, err)

	_, _ = io.Copy(ioutil.Discard, resp.Body) // consume body to complete request handler
	resp.Body.Close()

	if wg != nil {
		wg.Wait()
	}

	tracer.Flush(nil)
	payloads := transport.Payloads()

	count := 0
	for len(payloads.Transactions) == 0 {
		// the transaction is ended after the response body is fully written,
		// so a single call to `tracer.Flush()` may not have an event yet
		// enqueued. Continue to call `tracer.Flush()` and wait for the
		// transaction to have ended.
		// This is unique to fasthttp's implementation.
		if count > 10 {
			t.Fatal("no transactions found")
		}

		count++

		time.Sleep(100 * time.Millisecond)
		tracer.Flush(nil)
		payloads = transport.Payloads()
	}

	assertFn(payloads.Transactions[0], resp)

	// s.Serve returns on ln.Close()
	ln.Close()
	require.NoError(t, <-shutdown)
}

func TestServerHTTPResponse(t *testing.T) {
	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			ctx.Error(fasthttp.StatusMessage(fasthttp.StatusUnauthorized), fasthttp.StatusUnauthorized)
		},
		Name: "test-server",
	}

	assertFn := func(transaction model.Transaction, resp *http.Response) {
		expectedHeaders := model.Headers{
			{Key: "Content-Length", Values: []string{"12"}},
			{Key: "Content-Type", Values: []string{"text/plain; charset=utf-8"}},
			{Key: "Server", Values: []string{s.Name}},
		}

		assert.Equal(t, fasthttp.StatusUnauthorized, resp.StatusCode)
		assert.Equal(t, resp.StatusCode, transaction.Context.Response.StatusCode)
		assert.Equal(t, "GET /", transaction.Name)
		assert.Equal(t, "request", transaction.Type)
		assert.Equal(t, "HTTP 4xx", transaction.Result)
		assert.Equal(t, expectedHeaders, transaction.Context.Response.Headers)
	}

	testServer(t, s, nil, assertFn)
}

func TestServerHTTPResponseStream(t *testing.T) {
	streamResponseDuration := time.Millisecond * 100

	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
				w.WriteString("Hello world")
				time.Sleep(streamResponseDuration)
			})
		},
		Name: "test-server",
	}

	assertFn := func(transaction model.Transaction, resp *http.Response) {
		expectedHeaders := model.Headers{
			{Key: "Content-Type", Values: []string{"text/plain; charset=utf-8"}},
			{Key: "Server", Values: []string{s.Name}},
			{Key: "Transfer-Encoding", Values: []string{"chunked"}},
		}

		assert.Equal(t, fasthttp.StatusOK, resp.StatusCode)
		assert.Equal(t, resp.StatusCode, transaction.Context.Response.StatusCode)
		assert.Equal(t, "GET /", transaction.Name)
		assert.Equal(t, "request", transaction.Type)
		assert.Equal(t, "HTTP 2xx", transaction.Result)
		assert.GreaterOrEqual(t, transaction.Duration, float64(streamResponseDuration.Milliseconds()))
		assert.Equal(t, transaction.Context.Response.Headers, expectedHeaders)
	}

	testServer(t, s, nil, assertFn)
}

func TestServerHTTPResponseHijack(t *testing.T) {
	hijackResponseDuration := time.Millisecond * 100

	wg := new(sync.WaitGroup)
	wg.Add(1)

	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			ctx.Hijack(func(c net.Conn) {
				time.Sleep(hijackResponseDuration)
				wg.Done()
			})
		},
		Name: "test-server",
	}

	assertFn := func(transaction model.Transaction, resp *http.Response) {
		expectedHeaders := model.Headers{
			{Key: "Content-Length", Values: []string{"0"}},
			{Key: "Content-Type", Values: []string{"text/plain; charset=utf-8"}},
			{Key: "Server", Values: []string{s.Name}},
		}

		assert.Equal(t, fasthttp.StatusOK, resp.StatusCode)
		assert.Equal(t, resp.StatusCode, transaction.Context.Response.StatusCode)
		assert.Equal(t, "GET /", transaction.Name)
		assert.Equal(t, "request", transaction.Type)
		assert.Equal(t, "HTTP 2xx", transaction.Result)
		assert.GreaterOrEqual(t, transaction.Duration, float64(hijackResponseDuration.Milliseconds()))
		assert.Equal(t, transaction.Context.Response.Headers, expectedHeaders)
	}

	testServer(t, s, wg, assertFn)
}
