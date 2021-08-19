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

//go:build go1.9
// +build go1.9

package apmgocql // import "go.elastic.co/apm/module/apmgocql"

import (
	"context"

	"github.com/gocql/gocql"

	"go.elastic.co/apm"
	"go.elastic.co/apm/stacktrace"
)

func init() {
	stacktrace.RegisterLibraryPackage(
		"github.com/gocql",
		"github.com/gocassa",
	)
}

// Observer is a gocql.QueryObserver and gocql.BatchObserver,
// tracing operations and spans within a transaction.
type Observer struct {
	tracer *apm.Tracer
}

// NewObserver returns a new Observer which creates spans for
// observed gocql queries.
func NewObserver(o ...Option) *Observer {
	opts := options{
		tracer: apm.DefaultTracer,
	}
	for _, o := range o {
		o(&opts)
	}
	return &Observer{tracer: opts.tracer}
}

// ObserveBatch observes batch executions, and creates spans for the
// batch, and sub-spans for each statement therein.
func (o *Observer) ObserveBatch(ctx context.Context, batch gocql.ObservedBatch) {
	batchSpan, ctx := apm.StartSpanOptions(ctx, "BATCH", "db.cassandra.batch", apm.SpanOptions{
		Start: batch.Start,
	})
	batchSpan.Duration = batch.End.Sub(batch.Start)
	batchSpan.Context.SetDatabase(apm.DatabaseSpanContext{
		Type:     "cassandra",
		Instance: batch.Keyspace,
	})
	defer batchSpan.End()

	for _, statement := range batch.Statements {
		span, _ := apm.StartSpanOptions(ctx, querySignature(statement), "db.cassandra.query", apm.SpanOptions{
			Start: batch.Start,
		})
		span.Duration = batchSpan.Duration
		span.Context.SetDatabase(apm.DatabaseSpanContext{
			Type:      "cassandra",
			Instance:  batch.Keyspace,
			Statement: statement,
		})
		span.End()
	}

	if e := apm.CaptureError(ctx, batch.Err); e != nil {
		e.Timestamp = batch.End
		e.Send()
	}
}

// ObserveQuery observes query results, and creates spans for them.
func (o *Observer) ObserveQuery(ctx context.Context, query gocql.ObservedQuery) {
	span, _ := apm.StartSpanOptions(ctx, querySignature(query.Statement), "db.cassandra.query", apm.SpanOptions{
		Start: query.Start,
	})
	span.Duration = query.End.Sub(query.Start)
	span.Context.SetDatabase(apm.DatabaseSpanContext{
		Type:      "cassandra",
		Instance:  query.Keyspace,
		Statement: query.Statement,
	})
	if e := apm.CaptureError(ctx, query.Err); e != nil {
		e.Timestamp = query.End
		e.Send()
	}
	span.End()
}

type options struct {
	tracer *apm.Tracer
}

// Option sets options for observers.
type Option func(*options)
