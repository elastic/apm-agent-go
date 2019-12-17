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
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

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
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Read and close the body, which will cause further
			// reads of the body to return an error. This should
			// *not* cause the BodyCapturer to discard the body;
			// see https://github.com/elastic/apm-agent-go/issues/568.
			ioutil.ReadAll(r.Body)
			r.Body.Close()
		}),
		apmhttp.WithTracer(tracer),
	)
	tx := testPostTransaction(h, tracer, transport, strings.NewReader("foo"))
	assert.Equal(t, &model.RequestBody{Raw: "foo"}, tx.Context.Request.Body)
}

func TestHandlerCaptureBodyConcurrency(t *testing.T) {
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	tracer.SetCaptureBody(apm.CaptureBodyTransactions)
	h := apmhttp.Wrap(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		apmhttp.WithTracer(tracer.Tracer),
	)
	server := httptest.NewServer(h)
	defer server.Close()

	var wg sync.WaitGroup
	const N = 200
	sentBodies := make([]string, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sentBodies[i] = fmt.Sprint(i)
			for {
				req, _ := http.NewRequest("POST", server.URL+"/foo", strings.NewReader(sentBodies[i]))
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					// Guard against macOS test flakiness, where creating many concurrent
					// requests may lead to "connect: connection reset by peer".
					time.Sleep(10 * time.Millisecond)
					continue
				}
				resp.Body.Close()
				break
			}
		}(i)
	}
	wg.Wait()
	tracer.Flush(nil)

	transactions := tracer.Payloads().Transactions
	assert.Len(t, transactions, N)

	bodies := make([]string, N)
	for i, tx := range transactions {
		bodies[i] = tx.Context.Request.Body.Raw
	}
	assert.ElementsMatch(t, sentBodies, bodies)
}

func TestHandlerCaptureBodyRawTruncated(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.SetCaptureBody(apm.CaptureBodyTransactions)
	h := apmhttp.Wrap(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Reading 1024 bytes will read enough
			// to form the truncated body for single
			// byte runes, but not for multi-byte.
			r.Body.Read(make([]byte, 1024))
		}),
		apmhttp.WithTracer(tracer),
	)

	// Run through once with single-byte runes, and once
	// with multi-byte runes. This will give us coverage
	// over all code paths for body capture.
	bodyChars := []string{"x", "世"}
	for _, bodyChar := range bodyChars {
		body := strings.Repeat(bodyChar, 1025)
		tx := testPostTransaction(h, tracer, transport, strings.NewReader(body))
		assert.Equal(t, &model.RequestBody{Raw: strings.Repeat(bodyChar, 1024)}, tx.Context.Request.Body)
		transport.ResetPayloads()
	}
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
	server := httptest.NewServer(h)
	defer server.Close()

	req, _ := http.NewRequest("POST", server.URL+"/foo", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	resp.Body.Close()
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

func TestHandlerWithPanicPropagation(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	h := apmhttp.Wrap(
		http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			panic("foo")
		}),
		apmhttp.WithTracer(tracer),
		apmhttp.WithPanicPropagation(),
	)

	recovery := recoveryMiddleware(http.StatusBadGateway)
	h = recovery(h)

	server := httptest.NewServer(h)
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

func TestHandlerWithPanicPropagationResponseCodeForwarding(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	h := apmhttp.Wrap(
		http.HandlerFunc(panicHandler),
		apmhttp.WithTracer(tracer),
		apmhttp.WithPanicPropagation(),
	)

	recovery := recoveryMiddleware(0)
	h = recovery(h)

	server := httptest.NewServer(h)
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
	}))

	const traceparentValue = "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"
	makeReq := func(headers ...string) *http.Request {
		req, _ := http.NewRequest("GET", "http://server.testing/foo", nil)
		for i := 0; i < len(headers); i += 2 {
			req.Header.Set(headers[i], headers[i+1])
		}
		return req
	}

	h := apmhttp.Wrap(mux, apmhttp.WithTracer(tracer))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, makeReq("Elastic-Apm-Traceparent", traceparentValue))
	h.ServeHTTP(w, makeReq("traceparent", traceparentValue))
	h.ServeHTTP(w, makeReq("Elastic-Apm-Traceparent", traceparentValue, "traceparent", "nonsense"))
	tracer.Flush(nil)

	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, 3)
	for _, transaction := range payloads.Transactions {
		assert.Equal(t, "0af7651916cd43dd8448eb211c80319c", apm.TraceID(transaction.TraceID).String())
		assert.Equal(t, "b7ad6b7169203331", apm.SpanID(transaction.ParentID).String())
		assert.NotZero(t, transaction.ID)
	}
}

func TestHandlerTracestateHeader(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/foo", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		tx := apm.TransactionFromContext(req.Context())
		w.Write([]byte(tx.TraceContext().State.String()))
	}))

	makeReq := func(tracestate ...string) *http.Request {
		req, _ := http.NewRequest("GET", "http://server.testing/foo", nil)
		req.Header.Set("Traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
		if len(tracestate) > 0 {
			req.Header["Tracestate"] = tracestate
		}
		return req
	}

	h := apmhttp.Wrap(mux, apmhttp.WithTracer(apmtest.DiscardTracer))
	w := httptest.NewRecorder()

	w.Body = new(bytes.Buffer)
	h.ServeHTTP(w, makeReq("a=b, c=d"))
	assert.Equal(t, "a=b,c=d", w.Body.String())

	w.Body = new(bytes.Buffer)
	h.ServeHTTP(w, makeReq("a=b", "c=d"))
	assert.Equal(t, "a=b,c=d", w.Body.String())

	w.Body = new(bytes.Buffer)
	h.ServeHTTP(w, makeReq("a=")) // invalid tracestate
	assert.Equal(t, "", w.Body.String())
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
