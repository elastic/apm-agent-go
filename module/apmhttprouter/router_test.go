package apmhttprouter_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmhttprouter"
	"go.elastic.co/apm/transport/transporttest"
)

func TestRouter(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()
	router := apmhttprouter.New(apmhttprouter.WithTracer(tracer))

	router.DELETE("/DELETE", func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {})
	router.GET("/GET", func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {})
	router.HEAD("/HEAD", func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {})
	router.OPTIONS("/OPTIONS", func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {})
	router.PATCH("/PATCH", func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {})
	router.POST("/POST", func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {})
	router.PUT("/PUT", func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {})

	w := httptest.NewRecorder()
	methods := []string{"DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"}
	for _, method := range methods {
		sendRequest(router, w, method, "/"+method)
	}
	tracer.Flush(nil)

	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, len(methods))
	names := transactionNames(payloads.Transactions)
	for _, method := range methods {
		assert.Contains(t, names, method+" /"+method)
	}

	// Test router.Handle.
	router.Handle("GET", "/handle", func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {})
	sendRequest(router, w, "GET", "/handle")
	tracer.Flush(nil)
	payloads = transport.Payloads()
	transaction := payloads.Transactions[len(methods)]
	assert.Equal(t, "GET /handle", transaction.Name)
}

func TestRouterHTTPHandler(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()
	router := apmhttprouter.New(apmhttprouter.WithTracer(tracer))

	router.Handler("GET", "/handler", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	router.HandlerFunc("GET", "/handlerfunc", func(http.ResponseWriter, *http.Request) {})

	w := httptest.NewRecorder()
	sendRequest(router, w, "GET", "/handler")
	sendRequest(router, w, "GET", "/handlerfunc")
	tracer.Flush(nil)
	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, 2)

	names := transactionNames(payloads.Transactions)
	assert.Contains(t, names, "GET /handler")
	assert.Contains(t, names, "GET /handlerfunc")
}

func transactionNames(transactions []model.Transaction) []string {
	names := make([]string, len(transactions))
	for i, tx := range transactions {
		names[i] = tx.Name
	}
	return names
}

func sendRequest(r *apmhttprouter.Router, w http.ResponseWriter, method, path string) {
	req, _ := http.NewRequest(method, "http://server.testing"+path, nil)
	r.ServeHTTP(w, req)
}
