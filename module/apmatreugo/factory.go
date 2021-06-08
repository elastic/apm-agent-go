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

// +build go1.12

package apmatreugo // import "go.elastic.co/apm/module/apmatreugo"

import (
	"github.com/savsgio/atreugo/v11"
	"github.com/valyala/fasthttp"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmfasthttp"
)

// New returns a factory instance.
func New(options ...Option) *Factory {
	f := new(Factory)

	for i := range options {
		options[i](f)
	}

	if f.tracer == nil {
		f.tracer = apm.DefaultTracer
	}

	if f.requestName == nil {
		f.requestName = func(ctx *atreugo.RequestCtx) string {
			return apmfasthttp.ServerRequestName(ctx.RequestCtx)
		}
	}

	if f.requestIgnorer == nil {
		requestIgnorer := apmfasthttp.NewDynamicServerRequestIgnorer(f.tracer)
		f.requestIgnorer = func(ctx *atreugo.RequestCtx) bool {
			return requestIgnorer(ctx.RequestCtx)
		}
	}

	if f.recovery == nil {
		recovery := apmfasthttp.NewTraceRecovery(f.tracer)
		f.recovery = func(ctx *atreugo.RequestCtx, tx *apm.Transaction, bc *apm.BodyCapturer, recovered interface{}) {
			recovery(ctx.RequestCtx, tx, bc, recovered)
		}
	}

	return f
}

// Middleware returns a middleware.
func (f *Factory) Middleware() atreugo.Middleware {
	return func(ctx *atreugo.RequestCtx) error {
		if !f.tracer.Recording() || f.requestIgnorer(ctx) {
			return ctx.Next()
		}

		tx, _, err := apmfasthttp.StartTransactionWithBody(f.tracer, f.requestName(ctx), ctx.RequestCtx)
		if err != nil {
			return ctx.ErrorResponse(err, fasthttp.StatusInternalServerError)
		}

		tx.Context.SetFramework("atreugo", "v11")

		return ctx.Next()
	}
}

// PanicView returns a panic view.
func (f *Factory) PanicView() atreugo.PanicView {
	return func(ctx *atreugo.RequestCtx, err interface{}) {
		if f.panicPropagation {
			defer panic(err)
		}

		// 500 status code will be set only for APM transaction
		// to allow other middleware to choose a different response code
		if ctx.Response.Header.StatusCode() == fasthttp.StatusOK {
			ctx.Response.Header.SetStatusCode(fasthttp.StatusInternalServerError)
		}

		tx := apm.TransactionFromContext(ctx)
		bc := apm.BodyCapturerFromContext(ctx)

		f.recovery(ctx, tx, bc, err)
	}
}

// WithTracer returns a Option which sets t as the tracer
// to use for tracing server requests.
func WithTracer(t *apm.Tracer) Option {
	if t == nil {
		panic("t == nil")
	}

	return func(f *Factory) {
		f.tracer = t
	}
}

// WithServerRequestName returns a Option which sets fn as the function
// to use to obtain the transaction name for the given server request.
func WithServerRequestName(fn RequestNameFunc) Option {
	if fn == nil {
		panic("fn == nil")
	}

	return func(f *Factory) {
		f.requestName = fn
	}
}

// WithServerRequestIgnorer returns a Option which sets fn as the
// function to use to determine whether or not a server request should
// be ignored. If request ignorer is nil, all requests will be reported.
func WithServerRequestIgnorer(fn RequestIgnorerFunc) Option {
	if fn == nil {
		panic("fn == nil")
	}

	return func(f *Factory) {
		f.requestIgnorer = fn
	}
}

// WithRecovery returns a Option which sets r as the recovery
// function to use for tracing server requests.
func WithRecovery(r RecoveryFunc) Option {
	if r == nil {
		panic("r == nil")
	}

	return func(f *Factory) {
		f.recovery = r
	}
}

// WithPanicPropagation returns a Option which enable panic propagation.
// Any panic will be recovered and recorded as an error in a transaction, then
// panic will be caused again.
func WithPanicPropagation() Option {
	return func(f *Factory) {
		f.panicPropagation = true
	}
}
