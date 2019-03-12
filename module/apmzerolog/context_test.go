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

package apmzerolog_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/module/apmzerolog"
)

func ExampleTraceContextHook() {
	handleRequest := func(w http.ResponseWriter, req *http.Request) {
		logger := zerolog.Ctx(req.Context()).Hook(apmzerolog.TraceContextHook(req.Context()))
		logger.Error().Msg("message")
	}
	http.HandleFunc("/", handleRequest)
}

func TestTraceContextHookNothing(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).Hook(apmzerolog.TraceContextHook(context.Background()))
	logger.Info().Msg("message")

	assert.Equal(t, "{\"level\":\"info\",\"message\":\"message\"}\n", buf.String())
}

func TestTraceContextHookTransaction(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	tx, _, _ := apmtest.WithTransaction(func(ctx context.Context) {
		logger := logger.Hook(apmzerolog.TraceContextHook(ctx))
		logger.Info().Msg("message")
	})

	assert.Equal(t, fmt.Sprintf(`
{"level":"info","trace.id":"%x","transaction.id":"%x","message":"message"}
`[1:], tx.TraceID, tx.ID),
		buf.String(),
	)
}

func TestTraceContextHookSpan(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	tx, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		span, ctx := apm.StartSpan(ctx, "name", "type")
		logger := logger.Hook(apmzerolog.TraceContextHook(ctx))
		logger.Info().Msg("message")
		span.End()
	})

	require.Len(t, spans, 1)
	assert.Equal(t, fmt.Sprintf(`
{"level":"info","trace.id":"%x","transaction.id":"%x","span.id":"%x","message":"message"}
`[1:], tx.TraceID, tx.ID, spans[0].ID),
		buf.String(),
	)
}
