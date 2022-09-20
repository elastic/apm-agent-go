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

package apmsql // import "go.elastic.co/apm/module/apmsql/v2"

import (
	"context"
	"database/sql/driver"
	"errors"

	"go.elastic.co/apm/v2"
)

var _ driver.Validator = (*conn)(nil)

func newConn(in driver.Conn, d *tracingDriver, dsnInfo DSNInfo) driver.Conn {
	conn := &conn{Conn: in, driver: d}
	conn.dsnInfo = dsnInfo
	conn.namedValueChecker, _ = in.(namedValueChecker)
	conn.pinger, _ = in.(driver.Pinger)
	conn.queryer, _ = in.(driver.Queryer)
	conn.queryerContext, _ = in.(driver.QueryerContext)
	conn.connPrepareContext, _ = in.(driver.ConnPrepareContext)
	conn.execer, _ = in.(driver.Execer)
	conn.execerContext, _ = in.(driver.ExecerContext)
	conn.connBeginTx, _ = in.(driver.ConnBeginTx)
	conn.sessionResetter, _ = in.(driver.SessionResetter)
	conn.validator, _ = in.(driver.Validator)
	if in, ok := in.(driver.ConnBeginTx); ok {
		return &connBeginTx{conn, in}
	}
	return conn
}

type conn struct {
	driver.Conn
	driver  *tracingDriver
	dsnInfo DSNInfo

	namedValueChecker  namedValueChecker
	pinger             driver.Pinger
	queryer            driver.Queryer
	queryerContext     driver.QueryerContext
	connPrepareContext driver.ConnPrepareContext
	execer             driver.Execer
	execerContext      driver.ExecerContext
	connBeginTx        driver.ConnBeginTx
	sessionResetter    driver.SessionResetter
	validator          driver.Validator
}

func (c *conn) startStmtSpan(ctx context.Context, stmt, spanType string) (*apm.Span, context.Context) {
	return c.startSpan(ctx, c.driver.querySignature(stmt), spanType, stmt)
}

func (c *conn) startSpan(ctx context.Context, name, spanType, stmt string) (*apm.Span, context.Context) {
	span, ctx := apm.StartSpanOptions(ctx, name, spanType, apm.SpanOptions{
		ExitSpan: true,
	})
	if !span.Dropped() {
		if c.dsnInfo.Address != "" {
			span.Context.SetDestinationAddress(c.dsnInfo.Address, c.dsnInfo.Port)
			span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
				Name:     c.driver.driverName,
				Resource: c.driver.driverName,
			})
			span.Context.SetServiceTarget(apm.ServiceTargetSpanContext{
				Type: c.driver.driverName,
				Name: c.dsnInfo.Database,
			})
		}
		span.Context.SetDatabase(apm.DatabaseSpanContext{
			Instance:  c.dsnInfo.Database,
			Statement: stmt,
			Type:      "sql",
			User:      c.dsnInfo.User,
		})
	}
	return span, ctx
}

func (c *conn) finishSpan(ctx context.Context, span *apm.Span, result *driver.Result, resultError *error) {
	if *resultError == driver.ErrSkip {
		// TODO(axw) mark span as abandoned,
		// so it's not sent and not counted
		// in the span limit. Ideally remove
		// from the slice so memory is kept
		// in check.
		return
	}
	switch *resultError {
	case nil:
		if !span.Dropped() && result != nil && *result != nil && *result != driver.ResultNoRows {
			rowsAffected, err := (*result).RowsAffected()
			if err == nil && rowsAffected >= 0 {
				span.Context.SetDatabaseRowsAffected(rowsAffected)
			}
		}
	case driver.ErrBadConn, context.Canceled:
		// ErrBadConn is used by the connection pooling
		// logic in database/sql, and so is expected and
		// should not be reported.
		//
		// context.Canceled means the callers canceled
		// the operation, so this is also expected.
	default:
		if e := apm.CaptureError(ctx, *resultError); e != nil {
			e.Send()
		}
	}
	span.End()
}

func (c *conn) Ping(ctx context.Context) (resultError error) {
	if c.pinger == nil {
		return nil
	}
	span, ctx := c.startSpan(ctx, "ping", c.driver.pingSpanType, "")
	defer c.finishSpan(ctx, span, nil, &resultError)
	return c.pinger.Ping(ctx)
}

func (c *conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (_ driver.Rows, resultError error) {
	if c.queryerContext == nil && c.queryer == nil {
		return nil, driver.ErrSkip
	}
	span, ctx := c.startStmtSpan(ctx, query, c.driver.querySpanType)
	defer c.finishSpan(ctx, span, nil, &resultError)

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
	span, ctx := c.startStmtSpan(ctx, query, c.driver.prepareSpanType)
	defer c.finishSpan(ctx, span, nil, &resultError)
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

func (c *conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (result driver.Result, resultError error) {
	if c.execerContext == nil && c.execer == nil {
		return nil, driver.ErrSkip
	}
	span, ctx := c.startStmtSpan(ctx, query, c.driver.execSpanType)
	defer c.finishSpan(ctx, span, &result, &resultError)

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

func (c *conn) CheckNamedValue(nv *driver.NamedValue) error {
	return checkNamedValue(nv, c.namedValueChecker)
}

func (c *conn) ResetSession(ctx context.Context) error {
	if c.sessionResetter != nil {
		return c.sessionResetter.ResetSession(ctx)
	}
	return nil
}

func (c *conn) IsValid() bool {
	if c.validator != nil {
		return c.validator.IsValid()
	}
	return true
}

type connBeginTx struct {
	*conn
	connBeginTx driver.ConnBeginTx
}

func (c *connBeginTx) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	// TODO(axw) instrument commit/rollback?
	return c.connBeginTx.BeginTx(ctx, opts)
}
