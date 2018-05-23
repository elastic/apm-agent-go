package apmhttp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/context/ctxhttp"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/module/apmhttp"
)

func BenchmarkClient(b *testing.B) {
	b.Run("baseline", func(b *testing.B) {
		benchmarkClient(b, func(c *http.Client) *http.Client {
			return c
		})
	})
	b.Run("wrapped", func(b *testing.B) {
		benchmarkClient(b, func(c *http.Client) *http.Client {
			return apmhttp.WrapClient(c)
		})
	})
}

func benchmarkClient(b *testing.B, wrap func(*http.Client) *http.Client) {
	server := httptest.NewServer(testMux())
	defer server.Close()
	for _, path := range benchmarkPaths {
		b.Run(path, func(b *testing.B) {
			tracer := newTracer()
			defer tracer.Close()
			tx := tracer.StartTransaction("name", "type")
			ctx := elasticapm.ContextWithTransaction(context.Background(), tx)
			client := wrap(nil)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				resp, err := ctxhttp.Get(ctx, client, server.URL+path)
				require.NoError(b, err)
				resp.Body.Close()
			}
		})
	}
}
