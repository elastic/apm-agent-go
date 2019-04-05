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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmhttprouter"
	"go.elastic.co/apm/transport/transporttest"
)

func TestRouterHTTPSuite(t *testing.T) {
	tracer, recorder := transporttest.NewRecorderTracer()
	router := apmhttprouter.New(apmhttprouter.WithTracer(tracer))
	router.GET("/implicit_write", func(http.ResponseWriter, *http.Request, httprouter.Params) {})
	router.GET("/panic_before_write", func(http.ResponseWriter, *http.Request, httprouter.Params) {
		panic("boom")
	})
	router.GET("/panic_after_write", func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		w.Write([]byte("hello, world"))
		panic("boom")
	})
	suite.Run(t, &apmtest.HTTPTestSuite{
		Handler:  router,
		Tracer:   tracer,
		Recorder: recorder,
	})
}

func TestRouter(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()
	router := apmhttprouter.New(apmhttprouter.WithTracer(tracer))

	router.DELETE("/DELETE", func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {})
	router.GET("/GET", func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {})
	router.HEAD("/HEAD", func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {})
	router.OPTIONS("/OPTIONS", func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {})
	router.PATCH("/PATCH", func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {})
	router.POST("/POST", func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {})
	router.PUT("/PUT", func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {})

	w := httptest.NewRecorder()
	methods := []string{"DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"}
	for _, method := range methods {
		sendRequest(router, w, method, "/"+method)
	}
	tracer.Flush(nil)

	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, len(methods))
	names := transactionNames(payloads.Transactions)
	for _, method := range methods {
		assert.Contains(t, names, method+" /"+method)
	}

	// Test router.Handle.
	router.Handle("GET", "/handle", func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {})
	sendRequest(router, w, "GET", "/handle")
	tracer.Flush(nil)
	payloads = transport.Payloads()
	transaction := payloads.Transactions[len(methods)]
	assert.Equal(t, "GET /handle", transaction.Name)
}

func TestRouterHTTPHandler(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()
	router := apmhttprouter.New(apmhttprouter.WithTracer(tracer))

	router.Handler("GET", "/handler", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	router.HandlerFunc("GET", "/handlerfunc", func(http.ResponseWriter, *http.Request) {})

	w := httptest.NewRecorder()
	sendRequest(router, w, "GET", "/handler")
	sendRequest(router, w, "GET", "/handlerfunc")
	tracer.Flush(nil)
	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, 2)

	names := transactionNames(payloads.Transactions)
	assert.Contains(t, names, "GET /handler")
	assert.Contains(t, names, "GET /handlerfunc")
}

func TestRouterUnknownRoute(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()
	router := apmhttprouter.New(apmhttprouter.WithTracer(tracer))

	router.HandlerFunc("GET", "/foo", func(http.ResponseWriter, *http.Request) {})

	w := httptest.NewRecorder()
	sendRequest(router, w, "PUT", "/foo")
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

	w = httptest.NewRecorder()
	sendRequest(router, w, "GET", "/bar")
	assert.Equal(t, http.StatusNotFound, w.Code)

	tracer.Flush(nil)
	transactions := transport.Payloads().Transactions
	require.Len(t, transactions, 2)

	assert.Equal(t, "PUT unknown route", transactions[0].Name)
	assert.Equal(t, "GET unknown route", transactions[1].Name)
}

func transactionNames(transactions []model.Transaction) []string {
	names := make([]string, len(transactions))
	for i, tx := range transactions {
		names[i] = tx.Name
	}
	return names
}

func sendRequest(r *apmhttprouter.Router, w http.ResponseWriter, method, path string) {
	req, _ := http.NewRequest(method, "http://server.testing"+path, nil)
	r.ServeHTTP(w, req)
}
