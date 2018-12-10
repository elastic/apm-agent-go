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
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/transport/transporttest"
)

func TestMiddleware(t *testing.T) {
	handlers := beego.NewControllerRegister()
	handlers.Add("/thing/:id:int", &testController{}, "get:Get")
	apmbeego.AddFilters(handlers)

	tracer, transport := transporttest.NewRecorderTracer()
	server := httptest.NewServer(
		apmbeego.Middleware(apmhttp.WithTracer(tracer))(handlers),
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

func TestMiddlewareControllerPanic(t *testing.T) {
	handlers := beego.NewControllerRegister()
	handlers.Add("/thing/:id:int", &testController{}, "get:Get")
	apmbeego.AddFilters(handlers)

	tracer, transport := transporttest.NewRecorderTracer()
	server := httptest.NewServer(
		apmbeego.Middleware(apmhttp.WithTracer(tracer))(handlers),
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
