// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package apmhttp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/context/ctxhttp"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
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
			ctx := apm.ContextWithTransaction(context.Background(), tx)
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
