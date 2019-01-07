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
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/transport/transporttest"
)

func TestHandlerHTTPSuite(t *testing.T) {
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
	suite.Run(t, &apmtest.HTTPTestSuite{
		Handler:  apmhttp.Wrap(mux, apmhttp.WithTracer(tracer)),
		Tracer:   tracer,
		Recorder: recorder,
	})
}

func TestHandler(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	mux := http.NewServeMux()
	mux.Handle("/foo", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("bar"))
	}))

	h := apmhttp.Wrap(mux, apmhttp.WithTracer(tracer))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/foo", nil)
	req.Header.Set("User-Agent", "apmhttp_test")
	req.RemoteAddr = "client.testing:1234"
	h.ServeHTTP(w, req)
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

func TestHandlerCaptureBodyRaw(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.SetCaptureBody(apm.CaptureBodyTransactions)
	h := apmhttp.Wrap(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		apmhttp.WithTracer(tracer),
	)
	tx := testPostTransaction(h, tracer, transport, strings.NewReader("foo"))
	assert.Equal(t, &model.RequestBody{Raw: "foo"}, tx.Context.Request.Body)
}

func TestHandlerCaptureBodyForm(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.SetCaptureBody(apm.CaptureBodyTransactions)
	h := apmhttp.Wrap(
		http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
			if err := req.ParseForm(); err != nil {
				panic(err)
			}
		}),
		apmhttp.WithTracer(tracer),
	)
	tx := testPostTransaction(h, tracer, transport, strings.NewReader("foo=bar&foo=baz"))
	assert.Equal(t, &model.RequestBody{
		Form: url.Values{
			"foo": []string{"bar", "baz"},
		},
	}, tx.Context.Request.Body)
}

func TestHandlerCaptureBodyError(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.SetCaptureBody(apm.CaptureBodyAll)
	h := apmhttp.Wrap(
		http.HandlerFunc(panicHandler),
		apmhttp.WithTracer(tracer),
	)
	e := testPostError(h, tracer, transport, strings.NewReader("foo"))
	assert.Equal(t, &model.RequestBody{Raw: "foo"}, e.Context.Request.Body)
}

func TestHandlerCaptureBodyErrorIgnored(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.SetCaptureBody(apm.CaptureBodyTransactions)
	h := apmhttp.Wrap(
		http.HandlerFunc(panicHandler),
		apmhttp.WithTracer(tracer),
	)
	e := testPostError(h, tracer, transport, strings.NewReader("foo"))
	assert.Nil(t, e.Context.Request.Body) // only capturing for transactions
}

func testPostTransaction(h http.Handler, tracer *apm.Tracer, transport *transporttest.RecorderTransport, body io.Reader) model.Transaction {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "http://server.testing/foo", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(w, req)
	tracer.Flush(nil)
	return transport.Payloads().Transactions[0]
}

func testPostError(h http.Handler, tracer *apm.Tracer, transport *transporttest.RecorderTransport, body io.Reader) model.Error {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "http://server.testing/foo", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(w, req)
	tracer.Flush(nil)
	return transport.Payloads().Errors[0]
}

func TestHandlerRecovery(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	h := apmhttp.Wrap(
		http.HandlerFunc(panicHandler),
		apmhttp.WithTracer(tracer),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/foo", nil)
	h.ServeHTTP(w, req)
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

func TestHandlerRecoveryNoHeaders(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	h := apmhttp.Wrap(
		http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			panic("foo")
		}),
		apmhttp.WithTracer(tracer),
	)

	server := httptest.NewServer(h)
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

func TestHandlerRequestIgnorer(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	h := apmhttp.Wrap(
		http.NotFoundHandler(),
		apmhttp.WithTracer(tracer),
		apmhttp.WithServerRequestIgnorer(func(*http.Request) bool {
			return true
		}),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/foo", nil)
	h.ServeHTTP(w, req)
	tracer.Flush(nil)
	assert.Empty(t, transport.Payloads())
}

func TestHandlerTraceparentHeader(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	mux := http.NewServeMux()
	mux.Handle("/foo", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("bar"))
	}))

	h := apmhttp.Wrap(mux, apmhttp.WithTracer(tracer))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/foo", nil)
	req.Header.Set("Elastic-Apm-Traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	req.Header.Set("User-Agent", "apmhttp_test")
	req.RemoteAddr = "client.testing:1234"
	h.ServeHTTP(w, req)
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
