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

package apmgorilla_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmgorilla"
)

func TestMuxMiddleware(t *testing.T) {
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	r := mux.NewRouter()
	r.Use(apmgorilla.Middleware(apmgorilla.WithTracer(tracer.Tracer)))
	sub := r.PathPrefix("/prefix").Subrouter()
	sub.Path("/articles/{category}/{id:[0-9]+}").Handler(http.HandlerFunc(articleHandler))

	w := doRequest(r, "GET", "http://server.testing/prefix/articles/fiction/123?foo=123")
	assert.Equal(t, "fiction:123", w.Body.String())
	tracer.Flush(nil)

	payloads := tracer.Payloads()
	transaction := payloads.Transactions[0]

	assert.Equal(t, "GET /prefix/articles/{category}/{id}", transaction.Name)
	assert.Equal(t, "request", transaction.Type)
	assert.Equal(t, "HTTP 2xx", transaction.Result)

	assert.Equal(t, &model.Context{
		Request: &model.Request{
			Socket: &model.RequestSocket{
				RemoteAddress: "client.testing",
			},
			URL: model.URL{
				Full:     "http://server.testing/prefix/articles/fiction/123?foo=123",
				Protocol: "http",
				Hostname: "server.testing",
				Path:     "/prefix/articles/fiction/123",
				Search:   "foo=123",
			},
			Method:      "GET",
			HTTPVersion: "1.1",
		},
		Response: &model.Response{
			StatusCode: 200,
			Headers: model.Headers{{
				Key:    "Content-Type",
				Values: []string{"text/plain; charset=utf-8"},
			}},
		},
	}, transaction.Context)
}

func TestInstrumentUnknownRoute(t *testing.T) {
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	r := mux.NewRouter()
	apmgorilla.Instrument(r, apmgorilla.WithTracer(tracer.Tracer))
	r.HandleFunc("/foo", articleHandler).Methods("GET")

	w := doRequest(r, "GET", "http://server.testing/bar")
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = doRequest(r, "PUT", "http://server.testing/foo")
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	tracer.Flush(nil)

	transactions := tracer.Payloads().Transactions
	require.Len(t, transactions, 2)

	assert.Equal(t, "GET unknown route", transactions[0].Name)
	assert.Equal(t, "PUT unknown route", transactions[1].Name)
}

func articleHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	w.Write([]byte(fmt.Sprintf("%s:%s", vars["category"], vars["id"])))
}

func doRequest(h http.Handler, method, url string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, url, nil)
	req.RemoteAddr = "client.testing:1234"
	h.ServeHTTP(w, req)
	return w
}
