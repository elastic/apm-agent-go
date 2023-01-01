package apmpgxv5

import (
	"context"
	"github.com/jackc/pgx/v5"
	"go.elastic.co/apm/v2"
)

type spanType string

const (
	querySpanType   spanType = "db.postgresql.query"
	connectSpanType spanType = "db.postgresql.connect"
	copySpanType    spanType = "db.postgresql.copy"
	batchSpanType   spanType = "db.postgresql.batch"
)

const (
	databaseName           = "postgresql"
	databaseType           = "sql"
	destinationServiceType = "db"
)

var action = map[spanType]string{
	querySpanType:   "query",
	connectSpanType: "connect",
	copySpanType:    "copy",
	batchSpanType:   "batch",
}

func startSpan(ctx context.Context, name string, spanType spanType, conn *pgx.ConnConfig, opts apm.SpanOptions) (*apm.Span, context.Context, bool) {
	span, spanCtx := apm.StartSpanOptions(ctx, name, string(spanType), opts)

	// this line leads to panic, because apm.Tracer in trace is nil.
	// todo: fix this on review, idk how to fix it for now.
	//if span.Dropped() {
	//	span.End()
	//	return nil, nil, false
	//}

	if conn != nil {
		span.Context.SetDatabase(apm.DatabaseSpanContext{
			Instance:  conn.Database,
			Statement: name,
			Type:      databaseType,
			User:      conn.User,
		})

		span.Context.SetDestinationAddress(conn.Host, int(conn.Port))
	}

	span.Action = action[spanType]
	span.Context.SetServiceTarget(apm.ServiceTargetSpanContext{
		Name: databaseName,
		Type: destinationServiceType,
	})
	span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
		Name:     "",
		Resource: "postgresql",
	})

	return span, spanCtx, true
}

func endSpan(ctx context.Context, err error) {
	span := apm.SpanFromContext(ctx)
	defer span.End()

	if span.Dropped() {
		return
	}

	if err != nil {
		e := apm.CaptureError(ctx, err)
		e.SetSpan(span)
		e.Send()
	}

	return
}
