package apmsql

import (
	"context"
	"database/sql/driver"

	"go.elastic.co/apm"
)

func newStmt(in driver.Stmt, conn *conn, query string) driver.Stmt {
	stmt := &stmt{
		Stmt:      in,
		conn:      conn,
		signature: conn.driver.querySignature(query),
		query:     query,
	}
	stmt.columnConverter, _ = in.(driver.ColumnConverter)
	stmt.stmtExecContext, _ = in.(driver.StmtExecContext)
	stmt.stmtQueryContext, _ = in.(driver.StmtQueryContext)
	stmt.namedValueChecker, _ = in.(namedValueChecker)
	if stmt.namedValueChecker == nil {
		stmt.namedValueChecker = conn.namedValueChecker
	}
	return stmt
}

type stmt struct {
	driver.Stmt
	conn      *conn
	signature string
	query     string

	columnConverter   driver.ColumnConverter
	namedValueChecker namedValueChecker
	stmtExecContext   driver.StmtExecContext
	stmtQueryContext  driver.StmtQueryContext
}

func (s *stmt) startSpan(ctx context.Context, spanType string) (*apm.Span, context.Context) {
	return s.conn.startSpan(ctx, s.signature, spanType, s.query)
}

func (s *stmt) ColumnConverter(idx int) driver.ValueConverter {
	if s.columnConverter != nil {
		return s.columnConverter.ColumnConverter(idx)
	}
	return driver.DefaultParameterConverter
}

func (s *stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (_ driver.Result, resultError error) {
	span, ctx := s.startSpan(ctx, s.conn.driver.execSpanType)
	defer s.conn.finishSpan(ctx, span, &resultError)
	if s.stmtExecContext != nil {
		return s.stmtExecContext.ExecContext(ctx, args)
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
	return s.Exec(dargs)
}

func (s *stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (_ driver.Rows, resultError error) {
	span, ctx := s.startSpan(ctx, s.conn.driver.querySpanType)
	defer s.conn.finishSpan(ctx, span, &resultError)
	if s.stmtQueryContext != nil {
		return s.stmtQueryContext.QueryContext(ctx, args)
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
	return s.Query(dargs)
}

func (s *stmt) CheckNamedValue(nv *driver.NamedValue) error {
	return checkNamedValue(nv, s.namedValueChecker)
}
