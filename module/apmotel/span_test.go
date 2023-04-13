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
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"go.elastic.co/apm/v2"
	"go.elastic.co/apm/v2/model"
	"go.elastic.co/apm/v2/transport/transporttest"
)

func TestSpanEnd(t *testing.T) {
	for _, tt := range []struct {
		name    string
		getSpan func(context.Context, trace.Tracer) trace.Span

		expectedSpans        []model.Span
		expectedTransactions []model.Transaction
	}{
		{
			name: "with a root span",
			getSpan: func(ctx context.Context, tracer trace.Tracer) trace.Span {
				ctx, s := tracer.Start(ctx, "name")
				return s
			},
			expectedTransactions: []model.Transaction{
				{
					Name:    "name",
					Type:    "custom",
					Outcome: "success",
				},
			},
		},
		{
			name: "a root span with component attribute",
			getSpan: func(ctx context.Context, tracer trace.Tracer) trace.Span {
				ctx, s := tracer.Start(ctx, "name", trace.WithAttributes(attribute.String("component", "my_monolith")))
				return s
			},
			expectedTransactions: []model.Transaction{
				{
					Name:    "name",
					Type:    "my_monolith",
					Outcome: "success",
				},
			},
		},
		{
			name: "a root span with http attributes",
			getSpan: func(ctx context.Context, tracer trace.Tracer) trace.Span {
				ctx, s := tracer.Start(ctx, "name", trace.WithAttributes(
					attribute.String("http.method", "GET"),
					attribute.String("http.status_code", "404"),
					attribute.String("http.url", "/"),
				))
				return s
			},
			expectedTransactions: []model.Transaction{
				{
					Name:    "name",
					Type:    "request",
					Outcome: "success",
					Context: &model.Context{
						Request: &model.Request{
							URL: model.URL{
								Protocol: "http",
								Path:     "/",
							},
							Method:      "GET",
							HTTPVersion: "1.1",
						},
					},
				},
			},
		},
		{
			name: "a root span with unknown attribute",
			getSpan: func(ctx context.Context, tracer trace.Tracer) trace.Span {
				ctx, s := tracer.Start(ctx, "name", trace.WithAttributes(attribute.String("hello", "world")))
				return s
			},
			expectedTransactions: []model.Transaction{
				{
					Name:    "name",
					Type:    "custom",
					Outcome: "success",
					Context: &model.Context{
						Tags: model.IfaceMap{
							{
								Key:   "hello",
								Value: "world",
							},
						},
					},
				},
			},
		},
		{
			name: "with a child span",
			getSpan: func(ctx context.Context, tracer trace.Tracer) trace.Span {
				ctx, _ = tracer.Start(ctx, "parentSpan")
				ctx, s := tracer.Start(ctx, "childSpan")
				return s
			},
			expectedSpans: []model.Span{
				{
					Name:    "childSpan",
					Type:    "custom",
					Outcome: "success",
				},
			},
		},
		{
			name: "a child span with a component attribute",
			getSpan: func(ctx context.Context, tracer trace.Tracer) trace.Span {
				ctx, _ = tracer.Start(ctx, "parentSpan")
				ctx, s := tracer.Start(ctx, "name", trace.WithAttributes(attribute.String("component", "my_service")))
				return s
			},
			expectedSpans: []model.Span{
				{
					Name:    "name",
					Type:    "custom",
					Subtype: "my_service",
					Outcome: "success",
				},
			},
		},
		{
			name: "a child span with db attributes",
			getSpan: func(ctx context.Context, tracer trace.Tracer) trace.Span {
				ctx, _ = tracer.Start(ctx, "parentSpan")
				ctx, s := tracer.Start(ctx, "name", trace.WithAttributes(
					attribute.String("db.instance", "instance_42"),
					attribute.String("db.statement", "SELECT * FROM *;"),
					attribute.String("db.type", "query"),
					attribute.String("db.user", "root"),
				))
				return s
			},
			expectedSpans: []model.Span{
				{
					Name:    "name",
					Type:    "db",
					Subtype: "query",
					Outcome: "success",
					Context: &model.SpanContext{
						Database: &model.DatabaseSpanContext{
							Instance:  "instance_42",
							Statement: "SELECT * FROM *;",
							Type:      "query",
							User:      "root",
						},
					},
				},
			},
		},
		{
			name: "a child span with http attributes",
			getSpan: func(ctx context.Context, tracer trace.Tracer) trace.Span {
				ctx, _ = tracer.Start(ctx, "parentSpan")
				ctx, s := tracer.Start(ctx, "name", trace.WithAttributes(
					attribute.String("http.url", "https://example.com"),
					attribute.String("http.method", "GET"),
					attribute.String("http.host", "localhost"),
				))
				return s
			},
			expectedSpans: []model.Span{
				{
					Name:    "name",
					Type:    "external",
					Subtype: "http",
					Outcome: "success",
					Context: &model.SpanContext{
						Destination: &model.DestinationSpanContext{
							Address: "example.com",
							Port:    443,
							Service: &model.DestinationServiceSpanContext{
								Type:     "external",
								Name:     "https://example.com",
								Resource: "example.com:443",
							},
						},
						HTTP: &model.HTTPSpanContext{
							URL: &url.URL{
								Scheme: "https",
								Host:   "example.com",
								Path:   "/",
							},
						},
					},
				},
			},
		},
		{
			name: "a child span with unknown attribute",
			getSpan: func(ctx context.Context, tracer trace.Tracer) trace.Span {
				ctx, _ = tracer.Start(ctx, "parentSpan")
				ctx, s := tracer.Start(ctx, "name", trace.WithAttributes(attribute.String("hello", "world")))
				return s
			},
			expectedSpans: []model.Span{
				{
					Name:    "name",
					Type:    "custom",
					Outcome: "success",
					Context: &model.SpanContext{
						Tags: model.IfaceMap{
							{
								Key:   "hello",
								Value: "world",
							},
						},
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			apmTracer, recorder := transporttest.NewRecorderTracer()
			tp, err := NewTracerProvider(WithAPMTracer(apmTracer))
			assert.NoError(t, err)
			tracer := newTracer(tp.(*tracerProvider))

			ctx := context.Background()
			s := tt.getSpan(ctx, tracer)
			s.End()

			apmTracer.Flush(nil)
			payloads := recorder.Payloads()

			if tt.expectedSpans != nil {
				for i := range payloads.Spans {
					payloads.Spans[i].ID = model.SpanID{}
					payloads.Spans[i].TransactionID = model.SpanID{}
					payloads.Spans[i].ParentID = model.SpanID{}
					payloads.Spans[i].TraceID = model.TraceID{}
					payloads.Spans[i].SampleRate = nil
					payloads.Spans[i].Duration = 0
					payloads.Spans[i].Timestamp = model.Time{}
				}

				assert.Equal(t, tt.expectedSpans, payloads.Spans)
			}
			if tt.expectedTransactions != nil {
				for i := range payloads.Transactions {
					payloads.Transactions[i].ID = model.SpanID{}
					payloads.Transactions[i].TraceID = model.TraceID{}
					payloads.Transactions[i].SampleRate = nil
					payloads.Transactions[i].Duration = 0
					payloads.Transactions[i].Timestamp = model.Time{}
				}

				assert.Equal(t, tt.expectedTransactions, payloads.Transactions)
			}
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

func TestSpanRecording(t *testing.T) {
	for _, tt := range []struct {
		name          string
		getSpan       func(context.Context, trace.Tracer) trace.Span
		wantRecording bool
	}{
		{
			name: "with a sampled root span",
			getSpan: func(ctx context.Context, tracer trace.Tracer) trace.Span {
				ctx, s := tracer.Start(ctx, "name")
				return s
			},
			wantRecording: true,
		},
		{
			name: "with a non-sampled root span",
			getSpan: func(ctx context.Context, tracer trace.Tracer) trace.Span {
				return &span{
					tx: &apm.Transaction{},
				}
			},
			wantRecording: false,
		},
		{
			name: "with a non-dropped child span",
			getSpan: func(ctx context.Context, tracer trace.Tracer) trace.Span {
				ctx, _ = tracer.Start(ctx, "parentSpan")
				ctx, s := tracer.Start(ctx, "childSpan")
				return s
			},
			wantRecording: true,
		},
		{
			name: "with a dropped child span",
			getSpan: func(ctx context.Context, tracer trace.Tracer) trace.Span {
				return &span{
					span: &apm.Span{},
				}
			},
			wantRecording: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tp, err := NewTracerProvider()
			assert.NoError(t, err)
			tracer := newTracer(tp.(*tracerProvider))

			ctx := context.Background()
			s := tt.getSpan(ctx, tracer)
			assert.Equal(t, tt.wantRecording, s.IsRecording())
		})
	}
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
