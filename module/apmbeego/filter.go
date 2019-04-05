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

package apmbeego

import (
	"context"
	"net/http"

	"github.com/astaxie/beego"
	beegocontext "github.com/astaxie/beego/context"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

type beegoFilterStateKey struct{}

type beegoFilterState struct {
	context *beegocontext.Context
}

func init() {
	AddFilters(beego.BeeApp.Handlers)
	WrapRecoverFunc(beego.BConfig)
}

// Middleware returns a beego.MiddleWare that traces requests and reports panics to Elastic APM.
func Middleware(o ...Option) func(http.Handler) http.Handler {
	opts := options{
		tracer: apm.DefaultTracer,
	}
	for _, o := range o {
		o(&opts)
	}
	return func(h http.Handler) http.Handler {
		return apmhttp.Wrap(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			tx := apm.TransactionFromContext(req.Context())
			if tx != nil {
				state := &beegoFilterState{}
				defer setTransactionContext(tx, state)
				ctx := context.WithValue(req.Context(), beegoFilterStateKey{}, state)
				req = apmhttp.RequestWithContext(ctx, req)
			}
			h.ServeHTTP(w, req)
		}), apmhttp.WithTracer(opts.tracer), apmhttp.WithServerRequestName(apmhttp.UnknownRouteRequestName))
	}
}

// AddFilters adds required filters to handlers.
//
// This is called automatically for the default app (beego.BeeApp),
// so if you beego.Router, beego.RunWithMiddleware, etc., then you
// do not need to call AddFilters.
func AddFilters(handlers *beego.ControllerRegister) {
	handlers.InsertFilter("*", beego.BeforeStatic, beforeStatic, false)
}

// WrapRecoverFunc updates config's RecoverFunc so that panics will be reported to Elastic APM
// for traced requests. For non-traced requests, the original RecoverFunc will be called.
//
// WrapRecoverFunc is called automatically for the global config, beego.BConfig.
func WrapRecoverFunc(config *beego.Config) {
	orig := config.RecoverFunc
	config.RecoverFunc = func(context *beegocontext.Context) {
		if tx := apm.TransactionFromContext(context.Request.Context()); tx == nil {
			orig(context)
		}
	}
}

func beforeStatic(context *beegocontext.Context) {
	state, ok := context.Request.Context().Value(beegoFilterStateKey{}).(*beegoFilterState)
	if ok {
		state.context = context
	}
}

func setTransactionContext(tx *apm.Transaction, state *beegoFilterState) {
	tx.Context.SetFramework("beego", beego.VERSION)
	if state.context != nil {
		if route, ok := state.context.Input.GetData("RouterPattern").(string); ok {
			tx.Name = state.context.Request.Method + " " + route
		}
	}
}

type options struct {
	tracer *apm.Tracer
}

// Option sets options for tracing.
type Option func(*options)

// WithTracer returns an Option which sets t as the tracer to use for tracing server requests.
func WithTracer(t *apm.Tracer) Option {
	if t == nil {
		panic("t == nil")
	}
	return func(o *options) {
		o.tracer = t
	}
}
