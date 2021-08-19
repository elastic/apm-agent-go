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

//go:build go1.12
// +build go1.12

package apmfasthttp // import "go.elastic.co/apm/module/apmfasthttp"

import (
	"github.com/valyala/fasthttp"

	"go.elastic.co/apm"
)

// Wrap returns a fasthttp.RequestHandler wrapping handler, reporting each request as
// a transaction to Elastic APM.
//
// By default, the returned RequestHandler will use apm.DefaultTracer.
// Use WithTracer to specify an alternative tracer.
//
// By default, the returned RequestHandler will recover panics, reporting
// them to the configured tracer. To override this behaviour, use
// WithRecovery.
func Wrap(handler fasthttp.RequestHandler, options ...ServerOption) fasthttp.RequestHandler {
	h := new(apmHandler)
	h.requestHandler = handler

	for i := range options {
		options[i](h)
	}

	if h.tracer == nil {
		h.tracer = apm.DefaultTracer
	}

	if h.requestName == nil {
		h.requestName = ServerRequestName
	}

	if h.requestIgnorer == nil {
		h.requestIgnorer = NewDynamicServerRequestIgnorer(h.tracer)
	}

	if h.recovery == nil {
		h.recovery = NewTraceRecovery(h.tracer)
	}

	return h.handler
}

func (h *apmHandler) handler(ctx *fasthttp.RequestCtx) {
	if !h.tracer.Recording() || h.requestIgnorer(ctx) {
		h.requestHandler(ctx)

		return
	}

	tx, bc, err := StartTransactionWithBody(h.tracer, h.requestName(ctx), ctx)
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)

		return
	}

	defer func() {
		if err := recover(); err != nil {
			if h.panicPropagation {
				defer panic(err)
			}

			// 500 status code will be set only for APM transaction
			// to allow other middleware to choose a different response code
			if ctx.Response.Header.StatusCode() == fasthttp.StatusOK {
				ctx.Response.Header.SetStatusCode(fasthttp.StatusInternalServerError)
			}

			h.recovery(ctx, tx, bc, err)
		}
	}()

	h.requestHandler(ctx)
}

// WithTracer returns a ServerOption which sets t as the tracer
// to use for tracing server requests.
func WithTracer(t *apm.Tracer) ServerOption {
	if t == nil {
		panic("t == nil")
	}

	return func(h *apmHandler) {
		h.tracer = t
	}
}

// WithServerRequestName returns a ServerOption which sets fn as the function
// to use to obtain the transaction name for the given server request.
func WithServerRequestName(fn RequestNameFunc) ServerOption {
	if fn == nil {
		panic("fn == nil")
	}

	return func(h *apmHandler) {
		h.requestName = fn
	}
}

// WithServerRequestIgnorer returns a ServerOption which sets fn as the
// function to use to determine whether or not a server request should
// be ignored. If request ignorer is nil, all requests will be reported.
func WithServerRequestIgnorer(fn RequestIgnorerFunc) ServerOption {
	if fn == nil {
		panic("fn == nil")
	}

	return func(h *apmHandler) {
		h.requestIgnorer = fn
	}
}

// WithRecovery returns a ServerOption which sets r as the recovery
// function to use for tracing server requests.
func WithRecovery(r RecoveryFunc) ServerOption {
	if r == nil {
		panic("r == nil")
	}
	return func(h *apmHandler) {
		h.recovery = r
	}
}

// WithPanicPropagation returns a ServerOption which enable panic propagation.
// Any panic will be recovered and recorded as an error in a transaction, then
// panic will be caused again.
func WithPanicPropagation() ServerOption {
	return func(h *apmHandler) {
		h.panicPropagation = true
	}
}
