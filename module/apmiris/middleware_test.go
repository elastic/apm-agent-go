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

package apmiris_test

import (
	"bytes"
	"github.com/kataras/iris/v12"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.elastic.co/apm/module/apmiris"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/transport/transporttest"
)

var (
	debugOutput bytes.Buffer
)

func init() {
	log.SetOutput(&debugOutput)
}

func TestMiddlewareHTTPSuite(t *testing.T) {
	tracer, recording := transporttest.NewRecorderTracer()
	e := iris.New()
	e.Use(apmiris.Middleware(e, apmiris.WithTracer(tracer)))

	e.Get("/implicit_write", func(c iris.Context) {})
	e.Get("/panic_before_write", func(c iris.Context) { panic("boom") })
	e.Get("/panic_after_write", func(c iris.Context) {
		c.StatusCode(http.StatusOK)
		c.Writef("hello word")
		panic("boom")
	})

	suite.Run(t, &apmtest.HTTPTestSuite{
		Handler:  e,
		Tracer:   tracer,
		Recorder: recording,
	})
}

func TestMiddleware(t *testing.T) {
	debugOutput.Reset()
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	e := iris.New()
	e.Use(apmiris.Middleware(e, apmiris.WithTracer(tracer)))

	e.Get("/hello/{name:string}", handleHello)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("Get", "http://server.testing/hello/isbel", nil)
	req.Header.Set("User-Agent", "apmiris_test")
	req.RemoteAddr = "client.testing:1234"
	e.ServeHTTP(w, req)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	transaction := payloads.Transactions[0]
	assert.Equal(t, "Get /hello/{name:string}", transaction.Name)
	assert.Equal(t, "request", transaction.Type)
	assert.Equal(t, "HTTP 2xx", transaction.Result)

	assert.Equal(t, &model.Context{
		Service: &model.Service{
			Framework: &model.Framework{
				Name:    "iris",
				Version: iris.Version,
			},
		},
		Request: &model.Request{
			Socket: &model.RequestSocket{
				RemoteAddress: "client.testing",
			},
			URL: model.URL{
				Full:     "http://server.testing/hello/isbel",
				Protocol: "http",
				Hostname: "server.testing",
				Path:     "/hello/isbel",
			},
			Method: "Get",
			Headers: model.Headers{{
				Key:    "User-Agent",
				Values: []string{"apmiris_test"},
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

func TestMiddlewareUnknownRoute(t *testing.T) {
	debugOutput.Reset()
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	e := iris.New()
	e.Use(apmiris.Middleware(e, apmiris.WithTracer(tracer)))

	w := doRequest(e, "PUT", "http://server.testing/foo")
	assert.Equal(t, http.StatusNotFound, w.Code)
	tracer.Flush(nil)

	transaction := transport.Payloads().Transactions[0]
	assert.Equal(t, "PUT unknown route", transaction.Name)
}

func TestMiddlewarePanic(t *testing.T) {
	debugOutput.Reset()
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	e := iris.New()
	e.Use(apmiris.Middleware(e, apmiris.WithTracer(tracer)))

	e.Get("/panic", handlePanic)

	w := doRequest(e, "Get", "http://server.testing/panic")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	tracer.Flush(nil)
	assertError(t, transport.Payloads(), "handlePanic", "boom", false)
}

func TestMiddlewarePanicHeadersSent(t *testing.T) {
	debugOutput.Reset()
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	e := iris.New()
	e.Use(apmiris.Middleware(e, apmiris.WithTracer(tracer)))

	e.Get("/panic", handlePanicAfterHeaders)

	w := doRequest(e, "Get", "http://server.testing/panic")
	assert.Equal(t, http.StatusOK, w.Code)
	tracer.Flush(nil)
	assertError(t, transport.Payloads(), "handlePanicAfterHeaders", "boom", false)

	// There should be no warnings about attempting to override
	// the status code after the headers were written.
	assert.NotContains(t, debugOutput.String(), "[WARNING] Headers were already written")
}

func TestMiddlewareError(t *testing.T) {
	debugOutput.Reset()
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	e := iris.New()
	e.Use(apmiris.Middleware(e, apmiris.WithTracer(tracer)))

	e.Get("/error", handleError)

	w := doRequest(e, "Get", "http://server.testing/error")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	tracer.Flush(nil)
	assertError(t, transport.Payloads(), "handleError", "wot", true)
}

func assertError(t *testing.T, payloads transporttest.Payloads, culprit, message string, handled bool) model.Error {
	error0 := payloads.Errors[0]

	require.NotNil(t, error0.Context)
	require.NotNil(t, error0.Exception)
	assert.NotEmpty(t, error0.TransactionID)
	assert.Equal(t, culprit, error0.Culprit)
	assert.Equal(t, message, error0.Exception.Message)
	assert.Equal(t, handled, error0.Exception.Handled)
	return error0
}

func handlePanic(c iris.Context) {
	panic("boom")
}

func handlePanicAfterHeaders(c iris.Context) {
	c.StatusCode(http.StatusOK)
	_, _ = c.JSON(iris.Map{
		"message": "wot",
	})

	panic("boom")
}

func handleError(c iris.Context) {
	c.StatusCode(http.StatusInternalServerError)
	_, _ = c.JSON(iris.Map{
		"message": "wot",
	})
}

func doRequest(e *iris.Application, method, url string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, url, nil)
	req.Header.Set("User-Agent", "apmiris_test")
	req.RemoteAddr = "client.testing:1234"
	e.ServeHTTP(w, req)
	return w
}
