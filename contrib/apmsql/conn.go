package apmsql

import (
	"context"
	"database/sql/driver"
	"errors"

	"github.com/elastic/apm-agent-go/contrib/apmsql/dsn"
	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/trace"
)

func newConn(in driver.Conn, d *tracingDriver, dsn_ string) driver.Conn {
	conn := &conn{Conn: in, driver: d}
	var dsnInfo dsn.Info
	if d.dsnParser != nil {
		dsnInfo = d.dsnParser(dsn_)
	}
	conn.spanContextBase.Database = &model.DatabaseSpanContext{
		Type:     "sql",
		Instance: dsnInfo.Database,
		User:     dsnInfo.User,
	}
	conn.pinger, _ = in.(driver.Pinger)
	conn.queryer, _ = in.(driver.Queryer)
	conn.queryerContext, _ = in.(driver.QueryerContext)
	conn.connPrepareContext, _ = in.(driver.ConnPrepareContext)
	conn.execer, _ = in.(driver.Execer)
	conn.execerContext, _ = in.(driver.ExecerContext)
	conn.connBeginTx, _ = in.(driver.ConnBeginTx)
	conn.sessionResetter, _ = in.(driver.SessionResetter)
	if in, ok := in.(driver.ConnBeginTx); ok {
		return &connBeginTx{conn, in}
	}
	return conn
}

type conn struct {
	driver.Conn
	driver          *tracingDriver
	spanContextBase model.SpanContext

	pinger             driver.Pinger
	queryer            driver.Queryer
	queryerContext     driver.QueryerContext
	connPrepareContext driver.ConnPrepareContext
	execer             driver.Execer
	execerContext      driver.ExecerContext
	connBeginTx        driver.ConnBeginTx
	sessionResetter    driver.SessionResetter
}

func (c *conn) finishSpan(ctx context.Context, span *trace.Span, query string, resultError error) {
	if resultError == driver.ErrSkip {
		// TODO(axw) mark span as abandoned,
		// so it's not sent and not counted
		// in the span limit. Ideally remove
		// from the slice so memory is kept
		// in check.
		return
	}
	if span.Name == "" {
		span.Name = c.driver.querySignature(query)
	}
	if span.Context == nil {
		span.Context = c.spanContext(query)
	}
	span.Done(-1)
	if e := trace.CaptureError(ctx, resultError); e != nil {
		if e.Exception.Stacktrace == nil {
			e.SetExceptionStacktrace(2)
		}
		e.Send()
	}
}

func (c *conn) spanContext(statement string) *model.SpanContext {
	spanContext := c.spanContextBase
	spanContext.Database.Statement = statement
	return &spanContext
}

func (c *conn) Ping(ctx context.Context) (resultError error) {
	if c.pinger == nil {
		return nil
	}
	span, ctx := trace.StartSpan(ctx, "ping", c.driver.spanType("ping"))
	if span != nil {
		defer c.finishSpan(ctx, span, "", resultError)
	}
	return c.pinger.Ping(ctx)
}

func (c *conn) ResetSession(ctx context.Context) error {
	if c.sessionResetter != nil {
		return c.sessionResetter.ResetSession(ctx)
	}
	return nil
}

func (c *conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (_ driver.Rows, resultError error) {
	if c.queryerContext == nil && c.queryer == nil {
		return nil, driver.ErrSkip
	}
	span, ctx := trace.StartSpan(ctx, "", c.driver.spanType("query"))
	if span != nil {
		defer c.finishSpan(ctx, span, query, resultError)
	}

	if c.queryerContext != nil {
		return c.queryerContext.QueryContext(ctx, query, args)
	}
	dargs, err := namedValueToValue(args)
	if err != nil {
		return nil, err
	}
	select {
	default:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return c.queryer.Query(query, dargs)
}

func (*conn) Query(query string, args []driver.Value) (driver.Rows, error) {
	return nil, errors.New("Query should never be called")
}

func (c *conn) PrepareContext(ctx context.Context, query string) (_ driver.Stmt, resultError error) {
	span, ctx := trace.StartSpan(ctx, "", c.driver.spanType("prepare"))
	if span != nil {
		defer c.finishSpan(ctx, span, query, resultError)
	}
	var stmt driver.Stmt
	var err error
	if c.connPrepareContext != nil {
		stmt, err = c.connPrepareContext.PrepareContext(ctx, query)
	} else {
		stmt, err = c.Prepare(query)
		if err == nil {
			select {
			default:
			case <-ctx.Done():
				stmt.Close()
				return nil, ctx.Err()
			}
		}
	}
	if stmt != nil {
		stmt = newStmt(stmt, c, query)
	}
	return stmt, err
}

func (c *conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (_ driver.Result, resultError error) {
	if c.execerContext == nil && c.execer == nil {
		return nil, driver.ErrSkip
	}
	span, ctx := trace.StartSpan(ctx, "", c.driver.spanType("exec"))
	if span != nil {
		defer c.finishSpan(ctx, span, query, resultError)
	}

	if c.execerContext != nil {
		return c.execerContext.ExecContext(ctx, query, args)
	}
	dargs, err := namedValueToValue(args)
	if err != nil {
		return nil, err
	}
	select {
	default:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return c.execer.Exec(query, dargs)
}

func (*conn) Exec(query string, args []driver.Value) (driver.Result, error) {
	return nil, errors.New("Exec should never be called")
}

type connBeginTx struct {
	*conn
	connBeginTx driver.ConnBeginTx
}

func (c *connBeginTx) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	// TODO(axw) instrument commit/rollback?
	return c.connBeginTx.BeginTx(ctx, opts)
}
