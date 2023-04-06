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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func TestSpanEnd(t *testing.T) {
	for _, tt := range []struct {
		name    string
		getSpan func(context.Context, trace.Tracer) trace.Span
	}{
		{
			name: "with a root span",
			getSpan: func(ctx context.Context, tracer trace.Tracer) trace.Span {
				ctx, s := tracer.Start(ctx, "name")
				return s
			},
		},
		{
			name: "with a child span",
			getSpan: func(ctx context.Context, tracer trace.Tracer) trace.Span {
				ctx, _ = tracer.Start(ctx, "parentSpan")
				ctx, s := tracer.Start(ctx, "childSpan")
				return s
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tp, err := NewTracerProvider()
			assert.NoError(t, err)
			tracer := newTracer(tp.(*tracerProvider))

			ctx := context.Background()
			s := tt.getSpan(ctx, tracer)
			s.End()
		})
	}
}

func TestSpanAddEvent(t *testing.T) {
	tp, err := NewTracerProvider()
	assert.NoError(t, err)
	tracer := newTracer(tp.(*tracerProvider))
	_, s := tracer.Start(context.Background(), "mySpan")

	assert.Equal(t, []event(nil), s.(*span).events)

	now := time.Now()
	s.AddEvent("test", trace.WithTimestamp(now))
	assert.Equal(t, []event{
		event{
			Name: "test",
			Time: now,
		},
	}, s.(*span).events)
}

func TestSpanRecordError(t *testing.T) {
	tp, err := NewTracerProvider()
	assert.NoError(t, err)
	tracer := newTracer(tp.(*tracerProvider))
	_, s := tracer.Start(context.Background(), "mySpan")

	assert.Equal(t, []event(nil), s.(*span).events)

	now := time.Now()
	s.RecordError(errors.New("test"), trace.WithTimestamp(now))
	assert.Equal(t, []event{
		event{
			Name: "exception",
			Attributes: []attribute.KeyValue{
				attribute.String("exception.type", "*errors.errorString"),
				attribute.String("exception.message", "test"),
			},
			Time: now,
		},
	}, s.(*span).events)
}

func TestSpanSetStatus(t *testing.T) {
	tp, err := NewTracerProvider()
	assert.NoError(t, err)
	tracer := newTracer(tp.(*tracerProvider))
	_, s := tracer.Start(context.Background(), "mySpan")

	assert.Equal(t, status{Code: codes.Unset, Description: ""}, s.(*span).status)
	s.SetStatus(codes.Error, "error")
	assert.Equal(t, status{Code: codes.Error, Description: "error"}, s.(*span).status)

	s.SetStatus(codes.Ok, "")
	assert.Equal(t, status{Code: codes.Ok}, s.(*span).status)

	s.SetStatus(codes.Error, "error")
	assert.Equal(t, status{Code: codes.Ok}, s.(*span).status)
}

func TestSpanSetName(t *testing.T) {
	for _, tt := range []struct {
		name      string
		getSpan   func(context.Context, trace.Tracer) trace.Span
		checkName func(*testing.T, *span)
	}{
		{
			name: "with a root span",
			getSpan: func(ctx context.Context, tracer trace.Tracer) trace.Span {
				ctx, s := tracer.Start(ctx, "name")
				return s
			},

			checkName: func(t *testing.T, s *span) {
				assert.Equal(t, s.tx.Name, "newName")
			},
		},
		{
			name: "with a child span",
			getSpan: func(ctx context.Context, tracer trace.Tracer) trace.Span {
				ctx, _ = tracer.Start(ctx, "parentSpan")
				ctx, s := tracer.Start(ctx, "childSpan")
				return s
			},

			checkName: func(t *testing.T, s *span) {
				assert.Equal(t, s.tx.Name, "parentSpan")
				assert.Equal(t, s.span.Name, "newName")
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tp, err := NewTracerProvider()
			assert.NoError(t, err)
			tracer := newTracer(tp.(*tracerProvider))

			ctx := context.Background()
			s := tt.getSpan(ctx, tracer)

			s.SetName("newName")
		})
	}
}

func TestSpanSetAttributes(t *testing.T) {
	tp, err := NewTracerProvider()
	assert.NoError(t, err)
	tracer := newTracer(tp.(*tracerProvider))
	_, s := tracer.Start(context.Background(), "mySpan")

	assert.Equal(t, []attribute.KeyValue(nil), s.(*span).attributes)
	s.SetAttributes(
		attribute.String("string", "abc"),
		attribute.Int("int", 42),
	)
	assert.Equal(t, []attribute.KeyValue{
		attribute.String("string", "abc"),
		attribute.Int("int", 42),
	}, s.(*span).attributes)
}
