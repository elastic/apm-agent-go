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

package apmnegroni_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/urfave/negroni"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/transport/transporttest"

	"go.elastic.co/apm/module/apmnegroni"
)

func TestMiddlewareHTTPSuite(t *testing.T) {
	tracer, recorder := transporttest.NewRecorderTracer()
	mux := http.NewServeMux()
	mux.HandleFunc("/implicit_write", func(w http.ResponseWriter, req *http.Request) {})
	mux.HandleFunc("/panic_before_write", func(w http.ResponseWriter, req *http.Request) {
		panic("boom")
	})
	mux.HandleFunc("/panic_after_write", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("hello, world"))
		panic("boom")
	})

	n := negroni.New()
	n.Use(apmnegroni.Middleware(apmnegroni.WithTracer(tracer)))
	n.UseHandler(mux)

	suite.Run(t, &apmtest.HTTPTestSuite{
		Handler:  n,
		Tracer:   tracer,
		Recorder: recorder,
	})
}

func TestMiddleware(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	mux := http.NewServeMux()
	mux.Handle("/foo", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("bar"))
	}))
	n := negroni.New()
	n.Use(apmnegroni.Middleware(apmnegroni.WithTracer(tracer)))
	n.UseHandler(mux)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/foo", nil)
	req.Header.Set("User-Agent", "apmhttp_test")
	req.RemoteAddr = "client.testing:1234"
	n.ServeHTTP(w, req)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	transaction := payloads.Transactions[0]
	assert.Equal(t, "GET /foo", transaction.Name)
	assert.Equal(t, "request", transaction.Type)
	assert.Equal(t, "HTTP 4xx", transaction.Result)

	assert.Equal(t, &model.Context{
		Request: &model.Request{
			Socket: &model.RequestSocket{
				RemoteAddress: "client.testing",
			},
			URL: model.URL{
				Full:     "http://server.testing/foo",
				Protocol: "http",
				Hostname: "server.testing",
				Path:     "/foo",
			},
			Method: "GET",
			Headers: model.Headers{{
				Key:    "User-Agent",
				Values: []string{"apmhttp_test"},
			}},
			HTTPVersion: "1.1",
		},
		Response: &model.Response{
			StatusCode: 418,
		},
	}, transaction.Context)
}

func TestMiddlewareRecovery(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	n := negroni.New()
	n.Use(apmnegroni.Middleware(apmnegroni.WithTracer(tracer)))
	n.UseHandler(http.HandlerFunc(panicHandler))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/foo", nil)
	n.ServeHTTP(w, req)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	error0 := payloads.Errors[0]
	transaction := payloads.Transactions[0]

	assert.Equal(t, "panicHandler", error0.Culprit)
	assert.Equal(t, "foo", error0.Exception.Message)

	assert.Equal(t, &model.Response{
		StatusCode: 418,
	}, transaction.Context.Response)
}

func TestMiddlewareRecoveryNoHeaders(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	n := negroni.New()
	n.Use(apmnegroni.Middleware(apmnegroni.WithTracer(tracer)))
	n.UseHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		panic("foo")
	}))

	server := httptest.NewServer(n)
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	resp.Body.Close()

	// Panic is translated into a 500 response.
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	error0 := payloads.Errors[0]
	transaction := payloads.Transactions[0]

	assert.Equal(t, &model.Response{StatusCode: resp.StatusCode}, transaction.Context.Response)
	assert.Equal(t, &model.Response{StatusCode: resp.StatusCode}, error0.Context.Response)
}

func TestMiddlewareWithPanicPropagation(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	n := negroni.New()
	n.Use(apmnegroni.Middleware(apmnegroni.WithTracer(tracer),
		apmnegroni.WithPanicPropagation()))
	n.UseHandler(
		http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			panic("foo")
		}))

	recovery := recoveryMiddleware(http.StatusBadGateway)
	server := httptest.NewServer(recovery(n))
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	error0 := payloads.Errors[0]
	transaction := payloads.Transactions[0]

	assert.Equal(t, &model.Response{StatusCode: http.StatusInternalServerError}, transaction.Context.Response)
	assert.Equal(t, &model.Response{StatusCode: http.StatusInternalServerError}, error0.Context.Response)
}

func TestMiddlewareWithPanicPropagationResponseCodeForwarding(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	n := negroni.New()
	n.Use(apmnegroni.Middleware(apmnegroni.WithTracer(tracer),
		apmnegroni.WithPanicPropagation()))
	n.UseHandler(http.HandlerFunc(panicHandler))

	recovery := recoveryMiddleware(0)
	server := httptest.NewServer(recovery(n))
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, http.StatusTeapot, resp.StatusCode)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	error0 := payloads.Errors[0]
	transaction := payloads.Transactions[0]

	assert.Equal(t, &model.Response{StatusCode: resp.StatusCode}, transaction.Context.Response)
	assert.Equal(t, &model.Response{StatusCode: resp.StatusCode}, error0.Context.Response)
}

func TestMiddlewareRequestIgnorer(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	n := negroni.New()
	n.Use(apmnegroni.Middleware(apmnegroni.WithTracer(tracer),
		apmnegroni.WithServerRequestIgnorer(func(*http.Request) bool {
			return true
		})))
	n.UseHandler(http.NotFoundHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/foo", nil)
	n.ServeHTTP(w, req)
	tracer.Flush(nil)
	assert.Empty(t, transport.Payloads())
}

func TestMiddlewareTraceparentHeader(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	mux := http.NewServeMux()
	mux.Handle("/foo", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("bar"))
	}))
	n := negroni.New()
	n.Use(apmnegroni.Middleware(apmnegroni.WithTracer(tracer)))
	n.UseHandler(mux)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/foo", nil)
	req.Header.Set("Elastic-Apm-Traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	req.Header.Set("User-Agent", "apmhttp_test")
	req.RemoteAddr = "client.testing:1234"
	n.ServeHTTP(w, req)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	transaction := payloads.Transactions[0]
	assert.Equal(t, "0af7651916cd43dd8448eb211c80319c", apm.TraceID(transaction.TraceID).String())
	assert.Equal(t, "b7ad6b7169203331", apm.SpanID(transaction.ParentID).String())
	assert.NotZero(t, transaction.ID)
	assert.Equal(t, "GET /foo", transaction.Name)
	assert.Equal(t, "request", transaction.Type)
	assert.Equal(t, "HTTP 4xx", transaction.Result)
}

func panicHandler(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusTeapot)
	panic("foo")
}

func recoveryMiddleware(code int) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					if code == 0 {
						return
					}
					w.WriteHeader(code)
				}
			}()
			next.ServeHTTP(w, req)
		})
	}
}
