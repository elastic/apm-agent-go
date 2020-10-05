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

package apmgormv2

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmsql"
)

func scopeContext(scope *gorm.DB) (context.Context, bool) {
	if scope.Statement == nil {
		return nil, false
	}
	ctx := scope.Statement.Context
	return ctx, ctx != nil
}

func registerCallbacks(db *gorm.DB, dsnInfo apmsql.DSNInfo) {
	driverName := db.Dialector.Name()
	switch driverName {
	case "postgres":
		driverName = "postgresql"
	}
	spanTypePrefix := fmt.Sprintf("db.%s.", driverName)
	querySpanType := spanTypePrefix + "query"
	execSpanType := spanTypePrefix + "exec"

	const callbackPrefix = "elasticapm"

	// Create Callbacks
	_ = db.Callback().Create().
		Before("gorm:create").
		Register(fmt.Sprintf("%s:before:%s", callbackPrefix, execSpanType), newBeforeCallback(execSpanType))

	_ = db.Callback().Create().
		After("gorm:create").
		Register(fmt.Sprintf("%s:after:%s", callbackPrefix, execSpanType), newAfterCallback(dsnInfo))

	// Delete Callbacks
	_ = db.Callback().Delete().
		Before("gorm:delete").
		Register(fmt.Sprintf("%s:before:%s", callbackPrefix, execSpanType), newBeforeCallback(execSpanType))

	_ = db.Callback().Delete().
		After("gorm:delete").
		Register(fmt.Sprintf("%s:after:%s", callbackPrefix, execSpanType), newAfterCallback(dsnInfo))

	// Update Callbacks
	_ = db.Callback().Update().
		Before("gorm:update").
		Register(fmt.Sprintf("%s:before:%s", callbackPrefix, execSpanType), newBeforeCallback(execSpanType))

	_ = db.Callback().Update().
		After("gorm:update").
		Register(fmt.Sprintf("%s:after:%s", callbackPrefix, execSpanType), newAfterCallback(dsnInfo))

	// Query Callbacks
	_ = db.Callback().Query().
		Before("gorm:query").
		Register(fmt.Sprintf("%s:before:%s", callbackPrefix, querySpanType), newBeforeCallback(querySpanType))

	_ = db.Callback().Query().
		After("gorm:query").
		Register(fmt.Sprintf("%s:after:%s", callbackPrefix, querySpanType), newAfterCallback(dsnInfo))

	// Row Query Callbacks
	_ = db.Callback().Row().
		Before("gorm:row").
		Register(fmt.Sprintf("%s:before:%s", callbackPrefix, querySpanType), newBeforeCallback(querySpanType))

	_ = db.Callback().Row().
		After("gorm:row").
		Register(fmt.Sprintf("%s:after:%s", callbackPrefix, querySpanType), newAfterCallback(dsnInfo))

	// Raw Query Callbacks
	_ = db.Callback().Row().
		Before("gorm:raw").
		Register(fmt.Sprintf("%s:before:%s", callbackPrefix, querySpanType), newBeforeCallback(querySpanType))

	_ = db.Callback().Row().
		After("gorm:raw").
		Register(fmt.Sprintf("%s:after:%s", callbackPrefix, querySpanType), newAfterCallback(dsnInfo))
}

func newBeforeCallback(spanType string) func(*gorm.DB) {
	return func(scope *gorm.DB) {
		ctx, ok := scopeContext(scope)
		if !ok {
			return
		}
		span, ctx := apm.StartSpan(ctx, "", spanType)
		if span.Dropped() {
			span.End()
			ctx = nil
		}
		scope.Statement.Context = ctx
	}
}

func newAfterCallback(dsnInfo apmsql.DSNInfo) func(*gorm.DB) {
	return func(scope *gorm.DB) {
		ctx, ok := scopeContext(scope)
		if !ok {
			return
		}
		span := apm.SpanFromContext(ctx)
		if span == nil {
			return
		}

		statement := scope.Statement.SQL.String()
		span.Name = apmsql.QuerySignature(statement)
		span.Context.SetDestinationAddress(dsnInfo.Address, dsnInfo.Port)
		span.Context.SetDatabase(apm.DatabaseSpanContext{
			Instance:  dsnInfo.Database,
			Statement: statement,
			Type:      "sql",
			User:      dsnInfo.User,
		})
		defer span.End()

		// Capture errors, except for "record not found", which may be expected.
		for _, err := range extractErrors(scope) {
			if err == gorm.ErrRecordNotFound || err == sql.ErrNoRows {
				continue
			}
			if e := apm.CaptureError(ctx, err); e != nil {
				e.Send()
			}
		}
	}
}

func extractErrors(db *gorm.DB) []error {
	if db.Error == nil {
		return nil
	}
	var errs []error
	for _, err := range strings.Split(db.Error.Error(), "; ") {
		errs = append(errs, errors.New(err))
	}
	return errs
}
