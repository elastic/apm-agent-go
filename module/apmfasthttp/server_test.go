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

//go:build go1.12
// +build go1.12

package apmfasthttp_test

import (
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"go.elastic.co/apm/module/apmfasthttp"
	"go.elastic.co/apm/transport/transporttest"
)

func TestServerHTTPResponse(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	handler := func(ctx *fasthttp.RequestCtx) {
		ctx.Error(fasthttp.StatusMessage(fasthttp.StatusUnauthorized), fasthttp.StatusUnauthorized)
	}
	handler = apmfasthttp.Wrap(handler, apmfasthttp.WithTracer(tracer))
	shutdown := make(chan error)
	defer close(shutdown)
	ln, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)

	s := &fasthttp.Server{
		Handler: handler,
		Name:    "test-server",
	}

	go func() {
		shutdown <- s.Serve(ln)
	}()

	addr := ln.Addr().String()
	resp, err := http.Get("http://" + addr)
	require.NoError(t, err)
	io.Copy(ioutil.Discard, resp.Body) // consume body to complete request handler
	resp.Body.Close()
	assert.Equal(t, fasthttp.StatusUnauthorized, resp.StatusCode)
	tracer.Flush(nil)
	payloads := transport.Payloads()

	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			t.Fatal("timed out waiting for payload")
		default:
		}
		if len(payloads.Transactions) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
		tracer.Flush(nil)
		payloads = transport.Payloads()
	}

	transaction := payloads.Transactions[0]
	assert.Equal(t, "GET /", transaction.Name)
	assert.Equal(t, "request", transaction.Type)
	assert.Equal(t, "HTTP 4xx", transaction.Result)

	// s.Serve returns on ln.Close()
	ln.Close()
	require.NoError(t, <-shutdown)
}
