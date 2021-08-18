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

// +build go1.12

package apmfasthttp_test

import (
	"net"
	"net/http"
	"testing"

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

	// addr to make req to
	addr := ln.Addr().String()
	resp, err := http.Get("http://" + addr)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusUnauthorized, resp.StatusCode)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	transaction := payloads.Transactions[0]
	assert.Equal(t, "GET /", transaction.Name)
	assert.Equal(t, "request", transaction.Type)
	assert.Equal(t, "HTTP 4xx", transaction.Result)

	// s.Serve returns on ln.Close()
	ln.Close()
	require.NoError(t, <-shutdown)
}
