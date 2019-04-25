package apmbuffalo_test

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/runtime"
	"github.com/gobuffalo/x/sessions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmbuffalo"
	"go.elastic.co/apm/transport/transporttest"
)

var (
	debugOutput bytes.Buffer
)

func init() {
	log.SetOutput(&debugOutput)
}

func TestWithTracer_panics(t *testing.T) {
	assert.Panics(t, func() {
		apmbuffalo.WithTracer(nil)
	})
}

func TestMiddlewarePanic(t *testing.T) {
	debugOutput.Reset()
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	b := buffalo.New(buffalo.Options{
		SessionStore: sessions.Null{},
	})
	b.Use(apmbuffalo.Middleware(apmbuffalo.WithTracer(tracer)))
	b.GET("/panic", handlePanic)

	w := doRequest(b, "GET", "http://server.testing/panic")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	tracer.Flush(nil)
	assertError(t, transport.Payloads(), "boom", false)
}

func TestBuffaloMiddleware(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	b := buffalo.New(buffalo.Options{
		SessionStore: sessions.Null{},
	})
	b.Use(apmbuffalo.Middleware(apmbuffalo.WithTracer(tracer)))
	g := b.Group("/prefix")
	g.GET("/hello/{name}", handleHello)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/prefix/hello/isbel", nil)
	req.Header.Set("User-Agent", "apmbuffalo_test")
	req.RemoteAddr = "client.testing:1234"
	g.ServeHTTP(w, req)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	transaction := payloads.Transactions[0]
	assert.Equal(t, "GET /prefix/hello/{name}/", transaction.Name)
	assert.Equal(t, "request", transaction.Type)
	assert.Equal(t, "HTTP 4xx", transaction.Result)

	assert.Equal(t, &model.Context{
		Service: &model.Service{
			Framework: &model.Framework{
				Name:    "gobuffalo",
				Version: runtime.Version,
			},
		},
		Request: &model.Request{
			Socket: &model.RequestSocket{
				RemoteAddress: "client.testing",
			},
			URL: model.URL{
				Full:     "http://server.testing/prefix/hello/isbel/",
				Protocol: "http",
				Hostname: "server.testing",
				Path:     "/prefix/hello/isbel/",
			},
			Method: "GET",
			Headers: model.Headers{{
				Key:    "User-Agent",
				Values: []string{"apmbuffalo_test"},
			}},
			HTTPVersion: "1.1",
		},
		Response: &model.Response{
			StatusCode: 418,
			Headers: model.Headers{{
				Key:    "Content-Type",
				Values: []string{"text/plain; charset=utf-8"},
			}},
		},
	}, transaction.Context)
}

func TestBuffaloMiddlewareError(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	b := buffalo.New(buffalo.Options{
		SessionStore: sessions.Null{},
	})
	b.Use(apmbuffalo.Middleware(apmbuffalo.WithTracer(tracer)))
	b.GET("/error", handleError)

	w := doRequest(b, "GET", "http://server.testing/error")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	tracer.Flush(nil)
	assertError(t, transport.Payloads(), "wot", true)
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

func handleHello(c buffalo.Context) error {
	return c.Render(http.StatusTeapot, r.String(fmt.Sprintf("Hello, %s!", c.Param("name"))))
}

func handlePanic(c buffalo.Context) error {
	panic("boom")
}

func handleError(c buffalo.Context) error {
	return errors.New("wot")
}

func doRequest(a *buffalo.App, method, url string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, url, nil)
	req.Header.Set("User-Agent", "apmbuffalo_test")
	req.RemoteAddr = "client.testing:1234"
	a.ServeHTTP(w, req)
	return w
}
