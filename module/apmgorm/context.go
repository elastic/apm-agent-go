// +build go1.9

package apmgorm

import (
	"context"
	"fmt"

	"github.com/jinzhu/gorm"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/internal/sqlutil"
	"github.com/elastic/apm-agent-go/module/apmsql"
)

const (
	apmContextKey = "elasticapm:context"
)

// WithContext returns a copy of db with ctx recorded for use by
// the callbacks registered via RegisterCallbacks.
func WithContext(ctx context.Context, db *gorm.DB) *gorm.DB {
	return db.Set(apmContextKey, ctx)
}

func scopeContext(scope *gorm.Scope) (context.Context, bool) {
	value, ok := scope.Get(apmContextKey)
	if !ok {
		return nil, false
	}
	ctx, _ := value.(context.Context)
	return ctx, ctx != nil
}

// RegisterCallbacks registers callbacks on db for reporting spans
// to Elastic APM. This is called automatically by apmgorm.Open;
// it is provided for cases where a *gorm.DB is acquired by other
// means.
func RegisterCallbacks(db *gorm.DB) {
	registerCallbacks(db, apmsql.DSNInfo{})
}

func registerCallbacks(db *gorm.DB, dsnInfo apmsql.DSNInfo) {
	driverName := db.Dialect().GetName()
	switch driverName {
	case "postgres":
		driverName = "postgresql"
	}
	spanTypePrefix := fmt.Sprintf("db.%s.", driverName)
	querySpanType := spanTypePrefix + "query"
	execSpanType := spanTypePrefix + "exec"

	type params struct {
		spanType  string
		processor func() *gorm.CallbackProcessor
	}
	callbacks := map[string]params{
		"gorm:create": {
			spanType:  execSpanType,
			processor: func() *gorm.CallbackProcessor { return db.Callback().Create() },
		},
		"gorm:delete": {
			spanType:  execSpanType,
			processor: func() *gorm.CallbackProcessor { return db.Callback().Delete() },
		},
		"gorm:query": {
			spanType:  querySpanType,
			processor: func() *gorm.CallbackProcessor { return db.Callback().Query() },
		},
		"gorm:update": {
			spanType:  execSpanType,
			processor: func() *gorm.CallbackProcessor { return db.Callback().Update() },
		},
	}
	for name, params := range callbacks {
		const callbackPrefix = "elasticapm"
		params.processor().Before(name).Register(
			fmt.Sprintf("%s:before:%s", callbackPrefix, name),
			newBeforeCallback(params.spanType),
		)
		params.processor().After(name).Register(
			fmt.Sprintf("%s:after:%s", callbackPrefix, name),
			newAfterCallback(dsnInfo),
		)
	}
}

func newBeforeCallback(spanType string) func(*gorm.Scope) {
	return func(scope *gorm.Scope) {
		ctx, ok := scopeContext(scope)
		if !ok {
			return
		}
		span, ctx := elasticapm.StartSpan(ctx, "", spanType)
		if span.Dropped() {
			span.End()
			ctx = nil
		}
		scope.Set(apmContextKey, ctx)
	}
}

func newAfterCallback(dsnInfo apmsql.DSNInfo) func(*gorm.Scope) {
	return func(scope *gorm.Scope) {
		ctx, ok := scopeContext(scope)
		if !ok {
			return
		}
		span := elasticapm.SpanFromContext(ctx)
		if span == nil {
			return
		}
		span.Name = sqlutil.QuerySignature(scope.SQL)
		span.Context.SetDatabase(elasticapm.DatabaseSpanContext{
			Instance:  dsnInfo.Database,
			Statement: scope.SQL,
			Type:      "sql",
			User:      dsnInfo.User,
		})
		defer span.End()

		// Capture errors, except for "record not found", which may be expected.
		for _, err := range scope.DB().GetErrors() {
			if gorm.IsRecordNotFoundError(err) {
				continue
			}
			if e := elasticapm.CaptureError(ctx, err); e != nil {
				e.Send()
			}
		}
	}
}
