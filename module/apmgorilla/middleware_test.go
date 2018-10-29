package apmgorilla_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmgorilla"
	"go.elastic.co/apm/transport/transporttest"
)

func TestMuxMiddleware(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	r := mux.NewRouter()
	r.Use(apmgorilla.Middleware(apmgorilla.WithTracer(tracer)))
	sub := r.PathPrefix("/prefix").Subrouter()
	sub.Path("/articles/{category}/{id:[0-9]+}").Handler(http.HandlerFunc(articleHandler))

	w := doRequest(r, "GET", "http://server.testing/prefix/articles/fiction/123?foo=123")
	assert.Equal(t, "fiction:123", w.Body.String())
	tracer.Flush(nil)

	payloads := transport.Payloads()
	transaction := payloads.Transactions[0]

	assert.Equal(t, "GET /prefix/articles/{category}/{id}", transaction.Name)
	assert.Equal(t, "request", transaction.Type)
	assert.Equal(t, "HTTP 2xx", transaction.Result)

	assert.Equal(t, &model.Context{
		Request: &model.Request{
			Socket: &model.RequestSocket{
				RemoteAddress: "client.testing",
			},
			URL: model.URL{
				Full:     "http://server.testing/prefix/articles/fiction/123?foo=123",
				Protocol: "http",
				Hostname: "server.testing",
				Path:     "/prefix/articles/fiction/123",
				Search:   "foo=123",
			},
			Method: "GET",
			Headers: model.Headers{{
				Key:    "X-Real-Ip",
				Values: []string{"client.testing"},
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

func articleHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	w.Write([]byte(fmt.Sprintf("%s:%s", vars["category"], vars["id"])))
}

func doRequest(h http.Handler, method, url string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, url, nil)
	req.Header.Set("X-Real-IP", "client.testing")
	h.ServeHTTP(w, req)
	return w
}
