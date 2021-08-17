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

//go:build go1.9
// +build go1.9

package apmechov4_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.elastic.co/apm"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	apmecho "go.elastic.co/apm/module/apmechov4"
	"go.elastic.co/apm/transport/transporttest"
)

func TestMiddlewareHTTPSuite(t *testing.T) {
	tracer, recorder := transporttest.NewRecorderTracer()
	e := echo.New()
	e.Use(apmecho.Middleware(apmecho.WithTracer(tracer)))
	e.GET("/implicit_write", func(c echo.Context) error { return nil })
	e.GET("/panic_before_write", func(c echo.Context) error { panic("boom") })
	e.GET("/panic_after_write", func(c echo.Context) error {
		c.String(200, "hello, world")
		panic("boom")
	})
	e.POST("/explicit_error_capture", func(c echo.Context) error {
		ioutil.ReadAll(c.Request().Body)
		e := apm.CaptureError(c.Request().Context(), errors.New("total explosion"))
		e.Send()
		return c.String(503, e.Error())
	})
	suite.Run(t, &apmtest.HTTPTestSuite{
		Handler:  e,
		Tracer:   tracer,
		Recorder: recorder,
	})
}

func TestEchoMiddleware(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	e := echo.New()
	e.Use(apmecho.Middleware(apmecho.WithTracer(tracer)))
	e.GET("/hello/:name", handleHello)

	w := doRequest(e, "GET", "http://server.testing/hello/foo")
	assert.Equal(t, "Hello, foo!", w.Body.String())
	tracer.Flush(nil)

	payloads := transport.Payloads()
	transaction := payloads.Transactions[0]

	assert.Equal(t, "GET /hello/:name", transaction.Name)
	assert.Equal(t, "request", transaction.Type)
	assert.Equal(t, "HTTP 4xx", transaction.Result)

	assert.Equal(t, &model.Context{
		Service: &model.Service{
			Framework: &model.Framework{
				Name:    "echo",
				Version: echo.Version,
			},
		},
		Request: &model.Request{
			Socket: &model.RequestSocket{
				RemoteAddress: "client.testing",
			},
			URL: model.URL{
				Full:     "http://server.testing/hello/foo",
				Protocol: "http",
				Hostname: "server.testing",
				Path:     "/hello/foo",
			},
			Method:      "GET",
			HTTPVersion: "1.1",
			Headers: model.Headers{{
				Key:    "User-Agent",
				Values: []string{"apmecho_test"},
			}},
		},
		Response: &model.Response{
			StatusCode: 418,
			Headers: model.Headers{{
				Key:    "Content-Type",
				Values: []string{"text/plain; charset=UTF-8"},
			}},
		},
	}, transaction.Context)
}

func TestEchoMiddlewareUnknownRoute(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	e := echo.New()
	e.Use(apmecho.Middleware(apmecho.WithTracer(tracer)))
	e.GET("/hello/there", func(c echo.Context) error {
		return echo.ErrNotFound
	})

	doRequest(e, "GET", "http://server.testing/hello/there")
	doRequest(e, "PUT", "http://server.testing/hello/there")
	doRequest(e, "GET", "http://server.testing/ahoy/thar")
	doRequest(e, "PUT", "http://server.testing/ahoy/thar")

	// Replace echo.NotFoundHandler so the function pointers
	// don't match.
	oldHandler := echo.NotFoundHandler
	echo.NotFoundHandler = func(echo.Context) error {
		return echo.ErrNotFound
	}
	defer func() { echo.NotFoundHandler = oldHandler }()
	doRequest(e, "GET", "http://server.testing/ahoy/thar")

	tracer.Flush(nil)
	transactions := transport.Payloads().Transactions
	require.Len(t, transactions, 5)

	assert.Equal(t, "GET /hello/there", transactions[0].Name)
	assert.Equal(t, "PUT unknown route", transactions[1].Name)
	assert.Equal(t, "GET unknown route", transactions[2].Name)
	assert.Equal(t, "PUT unknown route", transactions[3].Name)
	assert.Equal(t, "GET unknown route", transactions[4].Name)
	for _, tx := range transactions {
		assert.Equal(t, "HTTP 4xx", tx.Result)
	}
}

func TestEchoMiddlewarePanic(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	e := echo.New()
	e.Use(apmecho.Middleware(apmecho.WithTracer(tracer)))
	e.GET("/panic", handlePanic)

	w := doRequest(e, "GET", "http://server.testing/panic")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	tracer.Flush(nil)
	assertError(t, transport.Payloads(), "handlePanic", "boom", false)
}

func TestEchoMiddlewarePanicHeadersSent(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	e := echo.New()
	e.Use(apmecho.Middleware(apmecho.WithTracer(tracer)))
	e.GET("/panic", handlePanicAfterHeaders)

	w := doRequest(e, "GET", "http://server.testing/panic")
	assert.Equal(t, http.StatusOK, w.Code)
	tracer.Flush(nil)
	assertError(t, transport.Payloads(), "handlePanicAfterHeaders", "boom", false)
}

func TestEchoMiddlewareError(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	e := echo.New()
	e.Use(apmecho.Middleware(apmecho.WithTracer(tracer)))
	e.GET("/error", handleError)

	w := doRequest(e, "GET", "http://server.testing/error")
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

func handleHello(c echo.Context) error {
	return c.String(http.StatusTeapot, fmt.Sprintf("Hello, %s!", c.Param("name")))
}

func handlePanic(c echo.Context) error {
	panic("boom")
}

func handlePanicAfterHeaders(c echo.Context) error {
	c.String(200, "")
	panic("boom")
}

func handleError(c echo.Context) error {
	return errors.New("wot")
}

func doRequest(e *echo.Echo, method, url string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, url, nil)
	req.Header.Set("User-Agent", "apmecho_test")
	req.RemoteAddr = "client.testing:1234"
	e.ServeHTTP(w, req)
	return w
}
