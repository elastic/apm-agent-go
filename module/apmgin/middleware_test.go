package apmgin_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/module/apmgin"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

func init() {
	// Make gin be quiet.
	gin.SetMode(gin.ReleaseMode)
}

func TestMiddleware(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	e := gin.New()
	e.Use(apmgin.Middleware(e, apmgin.WithTracer(tracer)))
	e.GET("/hello/:name", handleHello)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/hello/isbel", nil)
	req.Header.Set("User-Agent", "apmgin_test")
	req.RemoteAddr = "client.testing:1234"
	e.ServeHTTP(w, req)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	transaction := payloads.Transactions[0]
	assert.Equal(t, "GET /hello/:name", transaction.Name)
	assert.Equal(t, "request", transaction.Type)
	assert.Equal(t, "HTTP 2xx", transaction.Result)

	assert.Equal(t, &model.Context{
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
			Method: "GET",
			Headers: &model.RequestHeaders{
				UserAgent: "apmgin_test",
			},
			HTTPVersion: "1.1",
		},
		Response: &model.Response{
			StatusCode: 200,
			Headers: &model.ResponseHeaders{
				ContentType: "text/plain; charset=utf-8",
			},
		},
	}, transaction.Context)
}

func TestMiddlewarePanic(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	e := gin.New()
	e.Use(apmgin.Middleware(e, apmgin.WithTracer(tracer)))
	e.GET("/panic", handlePanic)

	w := doRequest(e, "GET", "http://server.testing/panic")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	tracer.Flush(nil)
	assertError(t, transport.Payloads(), "handlePanic", "boom", false)
}

func TestMiddlewarePanicHeadersSent(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	e := gin.New()
	e.Use(apmgin.Middleware(e, apmgin.WithTracer(tracer)))
	e.GET("/panic", handlePanicAfterHeaders)

	w := doRequest(e, "GET", "http://server.testing/panic")
	assert.Equal(t, http.StatusOK, w.Code)
	tracer.Flush(nil)
	assertError(t, transport.Payloads(), "handlePanicAfterHeaders", "boom", false)
}

func TestMiddlewareError(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	e := gin.New()
	e.Use(apmgin.Middleware(e, apmgin.WithTracer(tracer)))
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

func handlePanic(c *gin.Context) {
	panic("boom")
}

func handlePanicAfterHeaders(c *gin.Context) {
	c.String(200, "")
	panic("boom")
}

func handleError(c *gin.Context) {
	c.AbortWithError(500, errors.New("wot"))
}

func doRequest(e *gin.Engine, method, url string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, url, nil)
	req.Header.Set("User-Agent", "apmecho_test")
	req.RemoteAddr = "client.testing:1234"
	e.ServeHTTP(w, req)
	return w
}
