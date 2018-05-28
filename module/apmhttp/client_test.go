package apmhttp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context/ctxhttp"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/module/apmhttp"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

func TestClient(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	mux := http.NewServeMux()
	mux.Handle("/foo", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("bar"))
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	tx := tracer.StartTransaction("name", "type")
	ctx := elasticapm.ContextWithTransaction(context.Background(), tx)
	client := apmhttp.WrapClient(http.DefaultClient)
	resp, err := ctxhttp.Get(ctx, client, server.URL+"/foo")
	assert.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusTeapot, resp.StatusCode)
	tx.End()
	tracer.Flush(nil)

	payloads := transport.Payloads()
	require.Len(t, payloads, 1)
	transactions := payloads[0].Transactions()
	require.Len(t, transactions, 1)
	transaction := transactions[0]
	require.Len(t, transaction.Spans, 1)

	span := transaction.Spans[0]
	assert.Equal(t, "GET "+server.Listener.Addr().String(), span.Name)
	assert.Equal(t, "ext.http", span.Type)
	assert.Nil(t, span.Context)
}
