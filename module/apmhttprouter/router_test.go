package apmhttprouter_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/module/apmhttprouter"
	"github.com/elastic/apm-agent-go/transport/transporttest"
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
	require.Len(t, payloads, 1)
	transactions := payloads[0].Transactions()
	require.Len(t, transactions, len(methods))
	names := transactionNames(transactions)
	for _, method := range methods {
		assert.Contains(t, names, method+" /"+method)
	}

	// Test router.Handle.
	router.Handle("GET", "/handle", func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {})
	sendRequest(router, w, "GET", "/handle")
	tracer.Flush(nil)
	payloads = transport.Payloads()
	require.Len(t, payloads, 2)
	transactions = payloads[1].Transactions()
	require.Len(t, transactions, 1)
	assert.Equal(t, "GET /handle", transactions[0].Name)
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
	require.Len(t, payloads, 1)
	transactions := payloads[0].Transactions()
	require.Len(t, transactions, 2)

	names := transactionNames(transactions)
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
