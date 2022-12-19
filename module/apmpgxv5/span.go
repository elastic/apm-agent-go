package apmpgxv5

import (
	"context"
	"github.com/jackc/pgx/v5"
	"go.elastic.co/apm/v2"
	"time"
)

type spanType string

const (
	querySpanType   spanType = "db.postgresql.query"
	connectSpanType spanType = "db.postgresql.connect"
	copySpanType    spanType = "db.postgresql.copy"
	batchSpanType   spanType = "db.postgresql.batch"
)

var action = map[spanType]string{
	querySpanType:   "query",
	connectSpanType: "connect",
	copySpanType:    "copy",
	batchSpanType:   "batch",
}

func startSpan(ctx context.Context, name string, spanType spanType, conn *pgx.ConnConfig, opts apm.SpanOptions) (*apm.Span, context.Context, bool) {
	span, spanCtx := apm.StartSpanOptions(ctx, name, string(spanType), opts)

	// this line leads to panic (idk why)
	//if span.Dropped() {
	//	span.End()
	//	return nil, nil, false
	//}

	if conn != nil {
		span.Context.SetDatabase(apm.DatabaseSpanContext{
			Instance:  conn.Database,
			Statement: name,
			Type:      "sql",
			User:      conn.User,
		})

		span.Context.SetDestinationAddress(conn.Host, int(conn.Port))
	}

	span.Action = action[spanType]
	span.Context.SetServiceTarget(apm.ServiceTargetSpanContext{
		Name: "postgresql",
		Type: "db",
	})

	return span, spanCtx, true
}

func endSpan(ctx context.Context, data interface{}) {
	span := apm.SpanFromContext(ctx)
	defer span.End()

	if span.Dropped() {
		return
	}

	var apmErr error

	switch d := data.(type) {
	case pgx.TraceQueryEndData:
		apmErr = d.Err
	case pgx.TraceCopyFromEndData:
		apmErr = d.Err
	case pgx.TraceConnectEndData:
		apmErr = d.Err
	case pgx.TraceBatchEndData:
		apmErr = d.Err
	}

	if apmErr != nil {
		e := apm.CaptureError(ctx, apmErr)
		e.Timestamp = time.Now()
		e.SetSpan(span)
		e.Send()
	}

	return
}
