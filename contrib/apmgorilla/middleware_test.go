package apmgorilla_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/contrib/apmgorilla"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

func TestMuxMiddleware(t *testing.T) {
	tracer, transport := newRecordingTracer()
	defer tracer.Close()

	r := mux.NewRouter()
	r.Use(apmgorilla.Middleware(tracer))
	sub := r.PathPrefix("/prefix").Subrouter()
	sub.Path("/articles/{category}/{id:[0-9]+}").Handler(http.HandlerFunc(articleHandler))

	w := doRequest(r, "GET", "http://server.testing/prefix/articles/fiction/123?foo=123")
	assert.Equal(t, "fiction:123", w.Body.String())
	tracer.Flush(nil)

	payloads := transport.Payloads()
	require.Len(t, payloads, 1)
	assert.Contains(t, payloads[0], "transactions")

	transactions := payloads[0]["transactions"].([]interface{})
	require.Len(t, transactions, 1)
	transaction := transactions[0].(map[string]interface{})
	assert.Equal(t, "GET /prefix/articles/{category}/{id}", transaction["name"])
	assert.Equal(t, "request", transaction["type"])
	assert.Equal(t, "200", transaction["result"])

	context := transaction["context"].(map[string]interface{})
	assert.Equal(t, map[string]interface{}{
		"request": map[string]interface{}{
			"socket": map[string]interface{}{
				"remote_address": "client.testing",
			},
			"url": map[string]interface{}{
				"full":     "http://server.testing/prefix/articles/fiction/123?foo=123",
				"protocol": "http",
				"hostname": "server.testing",
				"pathname": "/prefix/articles/fiction/123",
				"search":   "foo=123",
			},
			"headers":      map[string]interface{}{},
			"method":       "GET",
			"http_version": "1.1",
		},
		"response": map[string]interface{}{
			"status_code":  float64(200),
			"finished":     true,
			"headers_sent": true,
			"headers": map[string]interface{}{
				"content-type": "text/plain; charset=utf-8",
			},
		},
	}, context)
}

func articleHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	w.Write([]byte(fmt.Sprintf("%s:%s", vars["category"], vars["id"])))
}

func newRecordingTracer() (*elasticapm.Tracer, *transporttest.RecorderTransport) {
	var transport transporttest.RecorderTransport
	tracer, err := elasticapm.NewTracer("apmgorilla_test", "0.1")
	if err != nil {
		panic(err)
	}
	tracer.Transport = &transport
	return tracer, &transport
}

func doRequest(h http.Handler, method, url string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, url, nil)
	req.Header.Set("X-Real-IP", "client.testing")
	h.ServeHTTP(w, req)
	return w
}
