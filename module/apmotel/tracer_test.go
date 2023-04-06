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

//go:build go1.18
// +build go1.18

package apmotel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

func TestTracerStartStoresSpanInContext(t *testing.T) {
	tp, err := NewTracerProvider()
	assert.NoError(t, err)
	tracer := newTracer(tp.(*tracerProvider))

	ctx := context.Background()
	ctx, s := tracer.Start(ctx, "name")

	assert.Equal(t, s, trace.SpanFromContext(ctx))
}

func TestTracerStartTransaction(t *testing.T) {
	tp, err := NewTracerProvider()
	assert.NoError(t, err)
	tracer := newTracer(tp.(*tracerProvider))

	ctx := context.Background()
	ctx, s := tracer.Start(ctx, "name")

	assert.NotNil(t, s.(*span).tx)
	assert.Nil(t, s.(*span).span)
}

func TestTracerStartChildSpan(t *testing.T) {
	tp, err := NewTracerProvider()
	assert.NoError(t, err)
	tracer := newTracer(tp.(*tracerProvider))

	ctx := context.Background()
	ctx, ps := tracer.Start(ctx, "parentSpan")
	ctx, cs := tracer.Start(ctx, "childSpan")

	assert.Equal(t, ps.(*span).tx, cs.(*span).tx)
	assert.NotNil(t, cs.(*span).span)
}
