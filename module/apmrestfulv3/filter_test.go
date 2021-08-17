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

//go:build go1.11
// +build go1.11

package apmrestfulv3_test

import (
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmrestfulv3"
	"go.elastic.co/apm/transport/transporttest"
)

func TestHandlerHTTPSuite(t *testing.T) {
	tracer, recorder := transporttest.NewRecorderTracer()
	var ws restful.WebService
	ws.Path("/").Consumes(restful.MIME_JSON, restful.MIME_XML, "application/x-www-form-urlencoded").Produces(restful.MIME_JSON, restful.MIME_XML)
	ws.Route(ws.GET("/implicit_write").To(func(req *restful.Request, resp *restful.Response) {}))
	ws.Route(ws.GET("/panic_before_write").To(func(req *restful.Request, resp *restful.Response) {
		panic("boom")
	}))
	ws.Route(ws.GET("/panic_after_write").To(func(req *restful.Request, resp *restful.Response) {
		resp.Write([]byte("hello, world"))
		panic("boom")
	}))
	ws.Route(ws.POST("/explicit_error_capture").To(func(req *restful.Request, resp *restful.Response) {
		ioutil.ReadAll(req.Request.Body)
		e := apm.CaptureError(req.Request.Context(), errors.New("total explosion"))
		e.Send()
		resp.WriteHeader(http.StatusServiceUnavailable)
		resp.Write([]byte(e.Error()))
	}))
	container := restful.NewContainer()
	container.Add(&ws)
	container.Filter(apmrestfulv3.Filter(apmrestfulv3.WithTracer(tracer)))

	suite.Run(t, &apmtest.HTTPTestSuite{
		Handler:  container,
		Tracer:   tracer,
		Recorder: recorder,
	})
}

func TestContainerFilter(t *testing.T) {
	type Thing struct {
		ID string
	}

	var ws restful.WebService
	ws.Path("/things").Consumes(restful.MIME_JSON, restful.MIME_XML).Produces(restful.MIME_JSON, restful.MIME_XML)
	ws.Route(ws.GET("/{id:[0-1]+}").To(func(req *restful.Request, resp *restful.Response) {
		if apm.TransactionFromContext(req.Request.Context()) == nil {
			panic("no transaction in context")
		}
		resp.WriteHeaderAndEntity(http.StatusTeapot, Thing{
			ID: req.PathParameter("id"),
		})
	}))

	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	container := restful.NewContainer()
	container.Add(&ws)
	container.Filter(apmrestfulv3.Filter(apmrestfulv3.WithTracer(tracer)))

	server := httptest.NewServer(container)
	defer server.Close()
	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	serverHost, serverPort, err := net.SplitHostPort(serverURL.Host)
	require.NoError(t, err)

	resp, err := http.Get(server.URL + "/things/123")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusTeapot, resp.StatusCode)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	assert.Len(t, payloads.Transactions, 1)
	transaction := payloads.Transactions[0]

	assert.Equal(t, "GET /things/{id}", transaction.Name)
	assert.Equal(t, "request", transaction.Type)
	assert.Equal(t, "HTTP 4xx", transaction.Result)

	assert.Equal(t, &model.Context{
		Service: &model.Service{
			Framework: &model.Framework{
				Name:    "go-restful",
				Version: "v3",
			},
		},
		Request: &model.Request{
			Socket: &model.RequestSocket{
				RemoteAddress: "127.0.0.1",
			},
			URL: model.URL{
				Full:     server.URL + "/things/123",
				Protocol: "http",
				Hostname: serverHost,
				Port:     serverPort,
				Path:     "/things/123",
			},
			Method:      "GET",
			HTTPVersion: "1.1",
			Headers: model.Headers{{
				Key:    "Accept-Encoding",
				Values: []string{"gzip"},
			}, {
				Key:    "User-Agent",
				Values: []string{"Go-http-client/1.1"},
			}},
		},
		Response: &model.Response{
			StatusCode: 418,
			Headers: model.Headers{{
				Key:    "Content-Type",
				Values: []string{"application/json"},
			}},
		},
	}, transaction.Context)
}

func TestContainerFilterPanic(t *testing.T) {
	var ws restful.WebService
	ws.Path("/things").Consumes(restful.MIME_JSON, restful.MIME_XML).Produces(restful.MIME_JSON, restful.MIME_XML)
	ws.Route(ws.GET("/{id}/foo").To(handlePanic))

	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	container := restful.NewContainer()
	container.Add(&ws)
	container.Filter(apmrestfulv3.Filter(apmrestfulv3.WithTracer(tracer)))

	server := httptest.NewServer(container)
	defer server.Close()
	resp, err := http.Get(server.URL + "/things/123/foo")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, 1)
	require.Len(t, payloads.Errors, 1)
	panicError := payloads.Errors[0]
	assert.Equal(t, payloads.Transactions[0].Context.Service, panicError.Context.Service)
	assert.Equal(t, payloads.Transactions[0].ID, panicError.ParentID)
	assert.Equal(t, "kablamo", panicError.Exception.Message)
	assert.Equal(t, "handlePanic", panicError.Culprit)
}

func handlePanic(req *restful.Request, resp *restful.Response) {
	panic("kablamo")
}

func TestContainerFilterUnknownRoute(t *testing.T) {
	var ws restful.WebService
	ws.Path("/things").Consumes(restful.MIME_JSON, restful.MIME_XML).Produces(restful.MIME_JSON, restful.MIME_XML)
	ws.Route(ws.GET("/{id}/foo").To(handlePanic))

	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	container := restful.NewContainer()
	container.Add(&ws)
	container.Filter(apmrestfulv3.Filter(apmrestfulv3.WithTracer(tracer)))

	server := httptest.NewServer(container)
	defer server.Close()
	resp, err := http.Get(server.URL + "/things/123/bar")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, 1)
	assert.Equal(t, "GET unknown route", payloads.Transactions[0].Name)
}
