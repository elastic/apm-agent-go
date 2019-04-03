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

package apmbeego_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/astaxie/beego"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmbeego"
	"go.elastic.co/apm/transport/transporttest"
)

func TestMiddleware(t *testing.T) {
	handlers := beego.NewControllerRegister()
	handlers.Add("/thing/:id:int", &testController{}, "get:Get")
	apmbeego.AddFilters(handlers)

	tracer, transport := transporttest.NewRecorderTracer()
	server := httptest.NewServer(
		apmbeego.Middleware(apmbeego.WithTracer(tracer))(handlers),
	)
	defer server.Close()

	resp, err := http.Get(server.URL + "/thing/1")
	require.NoError(t, err)
	defer resp.Body.Close()

	tracer.Flush(nil)
	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, 1)
	require.Len(t, payloads.Spans, 1)
	assert.Equal(t, "GET /thing/:id:int", payloads.Transactions[0].Name)
	assert.Equal(t, "HTTP 2xx", payloads.Transactions[0].Result)
	assert.Equal(t, "testController.Get", payloads.Spans[0].Name)
}

func TestMiddlewareUnknownRoute(t *testing.T) {
	handlers := beego.NewControllerRegister()
	handlers.Add("/thing/:id:int", &testController{}, "get:Get")
	apmbeego.AddFilters(handlers)

	tracer, transport := transporttest.NewRecorderTracer()
	server := httptest.NewServer(
		apmbeego.Middleware(apmbeego.WithTracer(tracer))(handlers),
	)
	defer server.Close()

	resp, err := http.Get(server.URL + "/zing/1")
	require.NoError(t, err)
	defer resp.Body.Close()

	tracer.Flush(nil)
	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, 1)
	require.Len(t, payloads.Spans, 0)
	assert.Equal(t, "GET unknown route", payloads.Transactions[0].Name)
	assert.Equal(t, "HTTP 4xx", payloads.Transactions[0].Result)
}

func TestMiddlewareControllerPanic(t *testing.T) {
	handlers := beego.NewControllerRegister()
	handlers.Add("/thing/:id:int", &testController{}, "get:Get")
	apmbeego.AddFilters(handlers)

	tracer, transport := transporttest.NewRecorderTracer()
	server := httptest.NewServer(
		apmbeego.Middleware(apmbeego.WithTracer(tracer))(handlers),
	)
	defer server.Close()

	resp, err := http.Get(server.URL + "/thing/666")
	require.NoError(t, err)
	defer resp.Body.Close()

	tracer.Flush(nil)
	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, 1)
	require.Len(t, payloads.Errors, 1)
	assert.Equal(t, "GET /thing/:id:int", payloads.Transactions[0].Name)
	assert.Equal(t, "HTTP 5xx", payloads.Transactions[0].Result)
	assert.Equal(t, "number of the beast", payloads.Errors[0].Exception.Message)
}

type testController struct {
	beego.Controller
}

func (c *testController) Get() {
	span, _ := apm.StartSpan(c.Ctx.Request.Context(), "testController.Get", "controller")
	defer span.End()

	id := c.Ctx.Input.Param(":id")
	if id == "666" {
		panic("number of the beast")
	}
	c.Ctx.Output.SetStatus(200)
}
