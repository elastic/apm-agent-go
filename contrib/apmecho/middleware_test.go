package apmecho_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/contrib/apmecho"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

func TestEchoMiddleware(t *testing.T) {
	tracer, transport := newRecordingTracer()
	defer tracer.Close()

	e := echo.New()
	e.Use(apmecho.Middleware(tracer))
	e.GET("/hello/:name", handleHello)

	w := doRequest(e, "GET", "http://server.testing/hello/foo")
	assert.Equal(t, "Hello, foo!", w.Body.String())
	tracer.Flush(nil)

	payloads := transport.Payloads()
	assert.Len(t, payloads, 1)
	assert.Contains(t, payloads[0], "transactions")

	transactions := payloads[0]["transactions"].([]interface{})
	assert.Len(t, transactions, 1)
	transaction := transactions[0].(map[string]interface{})
	assert.Equal(t, "GET /hello/:name", transaction["name"])
	assert.Equal(t, "request", transaction["type"])
	assert.Equal(t, "200", transaction["result"])

	context := transaction["context"].(map[string]interface{})
	assert.Equal(t, map[string]interface{}{
		"request": map[string]interface{}{
			"socket": map[string]interface{}{
				"remote_address": "client.testing",
			},
			"url": map[string]interface{}{
				"full":     "http://server.testing/hello/foo",
				"protocol": "http",
				"hostname": "server.testing",
				"pathname": "/hello/foo",
			},
			"method": "GET",
			"headers": map[string]interface{}{
				"user-agent": "apmecho_test",
			},
			"http_version": "1.1",
		},
		"response": map[string]interface{}{
			"status_code":  float64(200),
			"headers_sent": true,
			"headers": map[string]interface{}{
				"content-type": "text/plain; charset=UTF-8",
			},
		},
	}, context)
}

func TestEchoMiddlewarePanic(t *testing.T) {
	tracer, transport := newRecordingTracer()
	defer tracer.Close()

	e := echo.New()
	e.Use(apmecho.Middleware(tracer))
	e.GET("/panic", handlePanic)

	w := doRequest(e, "GET", "http://server.testing/panic")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	tracer.Flush(nil)
	assertError(t, transport.Payloads(), "handlePanic", "boom", false)
}

func TestEchoMiddlewareError(t *testing.T) {
	tracer, transport := newRecordingTracer()
	defer tracer.Close()

	e := echo.New()
	e.Use(apmecho.Middleware(tracer))
	e.GET("/error", handleError)

	w := doRequest(e, "GET", "http://server.testing/error")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	tracer.Flush(nil)
	assertError(t, transport.Payloads(), "handleError", "wot", true)
}

func assertError(t *testing.T, payloads []map[string]interface{}, culprit, message string, handled bool) {
	assert.Contains(t, payloads[0], "errors")
	errors := payloads[0]["errors"].([]interface{})
	error0 := errors[0].(map[string]interface{})

	assert.Contains(t, error0, "context")
	assert.Contains(t, error0, "transaction")
	assert.Contains(t, error0, "exception")
	assert.Equal(t, culprit, error0["culprit"])
	exception := error0["exception"].(map[string]interface{})
	assert.Equal(t, message, exception["message"])
	assert.Contains(t, exception, "stacktrace")
	assert.Equal(t, handled, exception["handled"])
}

func handleHello(c echo.Context) error {
	return c.String(http.StatusOK, fmt.Sprintf("Hello, %s!", c.Param("name")))
}

func handlePanic(c echo.Context) error {
	panic("boom")
}

func handleError(c echo.Context) error {
	return errors.New("wot")
}

func newRecordingTracer() (*elasticapm.Tracer, *transporttest.RecorderTransport) {
	var transport transporttest.RecorderTransport
	tracer, err := elasticapm.NewTracer("apmecho_test", "0.1")
	if err != nil {
		panic(err)
	}
	tracer.Transport = &transport
	return tracer, &transport
}

func doRequest(e *echo.Echo, method, url string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, url, nil)
	req.Header.Set("User-Agent", "apmecho_test")
	req.RemoteAddr = "client.testing:1234"
	e.ServeHTTP(w, req)
	return w
}
