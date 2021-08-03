package apmfiber_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	recoverMiddleware "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmfiber"
	"go.elastic.co/apm/transport/transporttest"
)

var (
	debugOutput bytes.Buffer
)

func init() {
	log.SetOutput(&debugOutput)
}

func TestMiddlewareMultipleSameHandler(t *testing.T) {
	debugOutput.Reset()

	do := func(url, method, targetTransactionName string) {
		tracer := apmtest.NewRecordingTracer()
		defer tracer.Close()

		app := fiber.New()
		app.Use(apmfiber.Middleware(apmfiber.WithTracer(tracer.Tracer)))
		app.Get("/admin/hello/:name", func(ctx *fiber.Ctx) error {
			return nil
		})
		app.Get("/consumer/hello/:name", func(ctx *fiber.Ctx) error {
			return ctx.SendString(ctx.Params("name"))
		})

		req, _ := http.NewRequestWithContext(context.TODO(), method, url, nil)
		req.Header.Set("User-Agent", "apmfiber_test")
		req.RemoteAddr = "client.testing:1234"
		_, _ = app.Test(req)
		tracer.Flush(nil)

		payloads := tracer.Payloads()
		transaction := payloads.Transactions[0]
		assert.Equal(t, targetTransactionName, transaction.Name)
	}

	for _, tc := range []struct {
		url             string
		method          string
		transactionName string
	}{
		{
			url:             "http://server.testing/admin/hello/isbel",
			method:          "GET",
			transactionName: "GET /admin/hello/:name",
		},
		{
			url:             "http://server.testing/consumer/hello/isbel",
			method:          "GET",
			transactionName: "GET /consumer/hello/:name",
		},
	} {
		do(tc.url, tc.method, tc.transactionName)
	}
}

func TestMiddleware(t *testing.T) {
	debugOutput.Reset()
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	e := fiber.New()
	e.Use(apmfiber.Middleware(apmfiber.WithTracer(tracer.Tracer)))
	e.Get("/hello/:name", handleHello)

	req, _ := http.NewRequestWithContext(context.TODO(), "GET", "http://server.testing/hello/isbel", nil)
	req.Header.Set("User-Agent", "apmfiber_test")
	req.RemoteAddr = "client.testing:1234"

	_, err := e.Test(req)
	assert.Nil(t, err)

	tracer.Flush(nil)

	payloads := tracer.Payloads()
	transaction := payloads.Transactions[0]
	assert.Equal(t, "GET /hello/:name", transaction.Name)
	assert.Equal(t, "request", transaction.Type)
	assert.Equal(t, "HTTP 2xx", transaction.Result)

	assert.Equal(t, &model.Context{
		Service: &model.Service{
			Framework: &model.Framework{
				Name:    "fiber",
				Version: fiber.Version,
			},
		},
		Request: &model.Request{
			Socket: &model.RequestSocket{
				Encrypted:     false,
				RemoteAddress: "remote-addr",
			},
			URL: model.URL{
				Full:     "http://server.testing/hello/isbel",
				Protocol: "http",
				Hostname: "server.testing",
				Path:     "/hello/isbel",
			},
			Method: "GET",
			Headers: model.Headers{
				{
					Key:    "Content-Length",
					Values: []string{"0"},
				},
				{
					Key:    "Host",
					Values: []string{"server.testing"},
				},
				{
					Key:    "User-Agent",
					Values: []string{"apmfiber_test"},
				},
			},
			HTTPVersion: "1.1",
		},
		Response: &model.Response{
			StatusCode: 200,
			Headers: model.Headers{
				{
					Key:    "Content-Type",
					Values: []string{"text/plain; charset=utf-8"},
				},
			},
		},
	}, transaction.Context)
}

func TestMiddlewareUnknownRoute(t *testing.T) {
	debugOutput.Reset()
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	e := fiber.New()
	e.Use(apmfiber.Middleware(apmfiber.WithTracer(tracer.Tracer)))

	req, err := http.NewRequestWithContext(context.TODO(), http.MethodPut, "http://server.testing/foo", nil)
	assert.Nil(t, err)

	resp, err := e.Test(req)
	assert.Nil(t, err)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	tracer.Flush(nil)

	transaction := tracer.Payloads().Transactions[0]
	assert.Equal(t, "PUT unknown route", transaction.Name)
}

func TestMiddlewareError(t *testing.T) {
	debugOutput.Reset()
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	e := fiber.New()
	e.Use(apmfiber.Middleware(apmfiber.WithTracer(tracer.Tracer)))
	e.Get("/error", handleError)

	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, "http://server.testing/error", nil)
	assert.Nil(t, err)

	resp, err := e.Test(req)
	assert.Nil(t, err)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	tracer.Flush(nil)
	assertError(t, tracer.Payloads(), "wot", true)
}

func TestMiddlewarePanic(t *testing.T) {
	debugOutput.Reset()
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	e := fiber.New()
	e.Use(apmfiber.Middleware(apmfiber.WithTracer(tracer.Tracer)), recoverMiddleware.New())
	e.Get("/panic", handlePanic)

	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, "http://server.testing/panic", nil)
	assert.Nil(t, err)

	resp, err := e.Test(req)
	assert.Nil(t, err)
	assert.Equal(t, resp.StatusCode, http.StatusInternalServerError)
}

func assertError(t *testing.T, payloads transporttest.Payloads, message string, handled bool) model.Error {
	error0 := payloads.Errors[0]

	require.NotNil(t, error0.Context)
	require.NotNil(t, error0.Exception)
	assert.NotEmpty(t, error0.TransactionID)
	assert.Equal(t, message, error0.Exception.Message)
	assert.Equal(t, handled, error0.Exception.Handled)

	return error0
}

func handlePanic(c *fiber.Ctx) error {
	panic("boom")
}

func handleError(c *fiber.Ctx) error {
	c.Status(500)

	return errors.New("wot")
}

func handleHello(c *fiber.Ctx) error {
	return c.Status(http.StatusOK).SendString(fmt.Sprintf("Hello, %s!", c.Params("name")))
}
