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

//go:build go1.14
// +build go1.14

package apmgopgv10 // import "go.elastic.co/apm/module/apmgopgv10"

import (
	"context"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmsql"
	"go.elastic.co/apm/stacktrace"
)

func init() {
	stacktrace.RegisterLibraryPackage("github.com/go-pg/pg/v10")
}

// Instrument modifies db such that operations are hooked and reported as spans
// to Elastic APM if they occur within the context of a captured transaction.
//
// If Instrument cannot instrument db, then an error will be returned.
func Instrument(db *pg.DB) error {
	db.AddQueryHook(&queryHook{})

	return nil
}

var _ pg.QueryHook = (*queryHook)(nil)

// queryHook is an implementation of pg.QueryHook that reports queries as spans to Elastic APM.
type queryHook struct{}

// BeforeQuery initiates the span for the database query
func (qh *queryHook) BeforeQuery(ctx context.Context, evt *pg.QueryEvent) (context.Context, error) {
	var (
		database string
		user     string
	)
	if db, ok := evt.DB.(*pg.DB); ok {
		opts := db.Options()
		user = opts.User
		database = opts.Database
	}

	sql, err := evt.UnformattedQuery()
	if err != nil {
		return ctx, errors.Wrap(err, "failed to generate sql")
	}

	span, ctx := apm.StartSpan(ctx, apmsql.QuerySignature(string(sql)), "db.postgresql.query")
	span.Context.SetDatabase(apm.DatabaseSpanContext{
		Statement: string(sql),

		// Static
		Type:     "sql",
		User:     user,
		Instance: database,
	})

	return ctx, nil
}

// AfterQuery ends the initiated span from BeforeQuery
func (qh *queryHook) AfterQuery(ctx context.Context, evt *pg.QueryEvent) error {
	if span := apm.SpanFromContext(ctx); span != nil {
		span.End()
	}

	return nil
}
