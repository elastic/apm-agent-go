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

// +build go1.9

package apmgopg

import (
	"fmt"

	"github.com/go-pg/pg"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmsql"
)

const elasticApmSpanKey = "go-apm-agent:span"

// QueryHook is an implementation of pg.QueryHook that reports queries as spans to Elastic APM.
type QueryHook struct{}

// BeforeQuery initiates the span for the database query
func (qh *QueryHook) BeforeQuery(evt *pg.QueryEvent) {
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
		// Expose the error making it a bit easier to debug
		sql = fmt.Sprintf("[go-pg] error: %s", err.Error())
	}

	span, _ := apm.StartSpan(evt.DB.Context(), apmsql.QuerySignature(sql), "db.postgresql.query")
	span.Context.SetDatabase(apm.DatabaseSpanContext{
		Statement: sql,

		// Static
		Type:     "sql",
		User:     user,
		Instance: database,
	})

	evt.Data[elasticApmSpanKey] = span
}

// AfterQuery ends the initiated span from BeforeQuery
func (qh *QueryHook) AfterQuery(evt *pg.QueryEvent) {
	span, ok := evt.Data[elasticApmSpanKey]
	if !ok {
		return
	}

	if s, ok := span.(*apm.Span); ok {
		s.End()
	}
}
