package apmgocql

import (
	"context"

	"github.com/gocql/gocql"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/stacktrace"
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
	tracer *elasticapm.Tracer
}

// NewObserver returns a new Observer which creates spans for
// observed gocql queries.
func NewObserver(o ...Option) *Observer {
	opts := options{
		tracer: elasticapm.DefaultTracer,
	}
	for _, o := range o {
		o(&opts)
	}
	return &Observer{tracer: opts.tracer}
}

// ObserveBatch observes batch executions, and creates spans for the
// batch, and sub-spans for each statement therein.
func (o *Observer) ObserveBatch(ctx context.Context, batch gocql.ObservedBatch) {
	batchSpan, ctx := elasticapm.StartSpan(ctx, "BATCH", "db.cassandra.batch")
	batchSpan.Timestamp = batch.Start
	batchSpan.Duration = batch.End.Sub(batch.Start)
	batchSpan.Context.SetDatabase(elasticapm.DatabaseSpanContext{
		Type:     "cassandra",
		Instance: batch.Keyspace,
	})
	defer batchSpan.End()

	for _, statement := range batch.Statements {
		span, _ := elasticapm.StartSpan(ctx, querySignature(statement), "db.cassandra.query")
		span.Timestamp = batchSpan.Timestamp
		span.Duration = batchSpan.Duration
		span.Context.SetDatabase(elasticapm.DatabaseSpanContext{
			Type:      "cassandra",
			Instance:  batch.Keyspace,
			Statement: statement,
		})
		span.End()
	}

	if e := elasticapm.CaptureError(ctx, batch.Err); e != nil {
		e.Timestamp = batch.End
		e.Send()
	}
}

// ObserveQuery observes query results, and creates spans for them.
func (o *Observer) ObserveQuery(ctx context.Context, query gocql.ObservedQuery) {
	span, _ := elasticapm.StartSpan(ctx, querySignature(query.Statement), "db.cassandra.query")
	span.Timestamp = query.Start
	span.Duration = query.End.Sub(query.Start)
	span.Context.SetDatabase(elasticapm.DatabaseSpanContext{
		Type:      "cassandra",
		Instance:  query.Keyspace,
		Statement: query.Statement,
	})
	span.End()

	if e := elasticapm.CaptureError(ctx, query.Err); e != nil {
		e.Timestamp = query.End
		e.Send()
	}
}

type options struct {
	tracer *elasticapm.Tracer
}

// Option sets options for observers.
type Option func(*options)
