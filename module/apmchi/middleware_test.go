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

package apmchi_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmchi"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/transport/transporttest"
)

func TestMiddleware(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	r := chi.NewRouter()
	r.Use(apmchi.Middleware(apmchi.WithTracer(tracer)))
	r.Route("/prefix", func(r chi.Router) {
		r.Get("/articles/{category}/{id}", articleHandler)
	})

	w := doRequest(r, "GET", "http://server.testing/prefix/articles/fiction/123?foo=123")
	assert.Equal(t, "fiction:123", w.Body.String())
	tracer.Flush(nil)

	payloads := transport.Payloads()
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
			Method: "GET",
			Headers: model.Headers{{
				Key:    "X-Real-Ip",
				Values: []string{"client.testing"},
			}},
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

func TestMiddleware_NestedUse(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	r := chi.NewRouter()
	r.Route("/prefix", func(r chi.Router) {
		r.Use(apmchi.Middleware(apmchi.WithTracer(tracer)))
		r.Get("/articles/{category}/{id}", articleHandler)
	})

	w := doRequest(r, "GET", "http://server.testing/prefix/articles/fiction/123?foo=123")
	assert.Equal(t, http.StatusOK, w.Code)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	transaction := payloads.Transactions[0]

	assert.Equal(t, "GET /prefix/articles/{category}/{id}", transaction.Name)
}

func TestMiddleware_With(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	r := chi.NewRouter()
	r.Route("/prefix", func(r chi.Router) {
		r.With(apmchi.Middleware(apmchi.WithTracer(tracer))).Get("/articles/{category}/{id}", articleHandler)
	})

	w := doRequest(r, "GET", "http://server.testing/prefix/articles/fiction/123?foo=123")
	assert.Equal(t, http.StatusOK, w.Code)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	transaction := payloads.Transactions[0]

	assert.Equal(t, "GET /prefix/articles/{category}/{id}", transaction.Name)
}

func TestMiddleware_NotFound(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	r := chi.NewRouter()
	r.Use(apmchi.Middleware(apmchi.WithTracer(tracer)))
	r.Route("/prefix", func(r chi.Router) {
		r.Get("/articles/{category}/{id}", articleHandler)
	})

	w := doRequest(r, "POST", "http://server.testing/bad/url")
	assert.Equal(t, http.StatusNotFound, w.Code)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	transaction := payloads.Transactions[0]

	assert.Equal(t, "POST unknown route", transaction.Name)
}

func TestWithTracer_panics(t *testing.T) {
	assert.Panics(t, func() {
		apmchi.WithTracer(nil)
	})
}

func TestWithRequestIgnorer(t *testing.T) {
	cases := []struct {
		name    string
		ignorer apmhttp.RequestIgnorerFunc
		expect  bool
	}{
		{
			"nil-ignorer",
			nil,
			true,
		},
		{
			"apmhttp.IgnoreNone",
			apmhttp.IgnoreNone,
			true,
		},
		{
			"apmhttp.NewRegexpRequestIgnorer",
			apmhttp.NewRegexpRequestIgnorer(regexp.MustCompile(".*")),
			false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			tracer, transport := transporttest.NewRecorderTracer()
			defer tracer.Close()

			r := chi.NewRouter()
			r.Use(apmchi.Middleware(
				apmchi.WithTracer(tracer),
				apmchi.WithRequestIgnorer(tt.ignorer),
			))
			r.Route("/prefix", func(r chi.Router) {
				r.Get("/articles/{category}/{id}", articleHandler)
			})

			w := doRequest(r, "POST", "http://server.testing/bad/url")
			assert.Equal(t, http.StatusNotFound, w.Code)
			tracer.Flush(nil)

			payloads := transport.Payloads()
			if tt.expect {
				assert.Equal(t, len(payloads.Transactions), 1)
			} else {
				assert.Equal(t, len(payloads.Transactions), 0)
			}
		})
	}
}

func articleHandler(w http.ResponseWriter, req *http.Request) {
	category := chi.URLParam(req, "category")
	id := chi.URLParam(req, "id")
	fmt.Fprintf(w, "%s:%s", category, id)
}

func doRequest(h http.Handler, method, url string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, url, nil)
	req.Header.Set("X-Real-IP", "client.testing")
	h.ServeHTTP(w, req)
	return w
}
