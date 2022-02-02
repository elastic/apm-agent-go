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

	"go.elastic.co/apm/v2"
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

func (s *stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (result driver.Result, resultError error) {
	span, ctx := s.startSpan(ctx, s.conn.driver.execSpanType)
	defer s.conn.finishSpan(ctx, span, &result, &resultError)
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
	defer s.conn.finishSpan(ctx, span, nil, &resultError)
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
