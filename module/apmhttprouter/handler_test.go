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

package apmhttprouter_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmhttprouter"
	"go.elastic.co/apm/transport/transporttest"
)

func TestWrap(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	router := httprouter.New()

	const route = "/hello/:name/go/*wild"
	router.GET(route, apmhttprouter.Wrap(
		func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
			w.WriteHeader(http.StatusTeapot)
			w.Write([]byte(fmt.Sprintf("%s:%s", p.ByName("name"), p.ByName("wild"))))
		},
		route,
		apmhttprouter.WithTracer(tracer),
	))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/hello/go/go/bananas", nil)
	req.Header.Set("User-Agent", "apmhttp_test")
	req.RemoteAddr = "client.testing:1234"
	router.ServeHTTP(w, req)
	tracer.Flush(nil)
	assert.Equal(t, "go:/bananas", w.Body.String())

	payloads := transport.Payloads()
	transaction := payloads.Transactions[0]
	assert.Equal(t, "GET /hello/:name/go/*wild", transaction.Name)
	assert.Equal(t, "request", transaction.Type)
	assert.Equal(t, "HTTP 4xx", transaction.Result)

	assert.Equal(t, &model.Context{
		Request: &model.Request{
			Socket: &model.RequestSocket{
				RemoteAddress: "client.testing",
			},
			URL: model.URL{
				Full:     "http://server.testing/hello/go/go/bananas",
				Protocol: "http",
				Hostname: "server.testing",
				Path:     "/hello/go/go/bananas",
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

func TestRecovery(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	router := httprouter.New()

	const route = "/panic"
	router.GET(route, apmhttprouter.Wrap(
		panicHandler, route,
		apmhttprouter.WithTracer(tracer),
	))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/panic", nil)
	router.ServeHTTP(w, req)
	tracer.Flush(nil)
	assert.Equal(t, http.StatusTeapot, w.Code)

	payloads := transport.Payloads()
	error0 := payloads.Errors[0]
	transaction := payloads.Transactions[0]

	assert.Equal(t, "panicHandler", error0.Culprit)
	assert.Equal(t, "foo", error0.Exception.Message)

	assert.Equal(t, &model.Response{
		StatusCode: 418,
	}, transaction.Context.Response)
}

func panicHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	w.WriteHeader(http.StatusTeapot)
	panic("foo")
}
