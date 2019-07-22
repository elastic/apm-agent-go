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

package apmhttprouter

import (
	"net/http"

	"github.com/julienschmidt/httprouter"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

// Wrap wraps h such that it will report requests as transactions
// to Elastic APM, using route in the transaction name.
//
// By default, the returned Handle will use apm.DefaultTracer.
// Use WithTracer to specify an alternative tracer.
//
// By default, the returned Handle will recover panics, reporting
// them to the configured tracer. To override this behaviour, use
// WithRecovery.
func Wrap(h httprouter.Handle, route string, o ...Option) httprouter.Handle {
	opts := gatherOptions(o...)
	return func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
		if !opts.tracer.Active() || opts.requestIgnorer(req) {
			h(w, req, p)
			return
		}
		tx, req := apmhttp.StartTransaction(opts.tracer, req.Method+" "+route, req)
		defer tx.End()

		body := opts.tracer.CaptureHTTPRequestBody(req)
		w, resp := apmhttp.WrapResponseWriter(w)
		defer func() {
			if v := recover(); v != nil {
				if resp.StatusCode == 0 {
					w.WriteHeader(http.StatusInternalServerError)
				}
				opts.recovery(w, req, resp, body, tx, v)
			}
			apmhttp.SetTransactionContext(tx, req, resp, body)
			body.Discard()
		}()
		h(w, req, p)
		if resp.StatusCode == 0 {
			resp.StatusCode = http.StatusOK
		}
	}
}

// WrapNotFoundHandler wraps h so that it is traced. If h is nil, then http.NotFoundHandler() will be used.
func WrapNotFoundHandler(h http.Handler, o ...Option) http.Handler {
	if h == nil {
		h = http.NotFoundHandler()
	}
	return wrapHandlerUnknownRoute(h, o...)
}

// WrapMethodNotAllowedHandler wraps h so that it is traced. If h is nil, then a default handler
// will be used that returns status code 405.
func WrapMethodNotAllowedHandler(h http.Handler, o ...Option) http.Handler {
	if h == nil {
		h = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		})
	}
	return wrapHandlerUnknownRoute(h, o...)
}

func wrapHandlerUnknownRoute(h http.Handler, o ...Option) http.Handler {
	opts := gatherOptions(o...)
	return apmhttp.Wrap(
		h,
		apmhttp.WithTracer(opts.tracer),
		apmhttp.WithRecovery(opts.recovery),
		apmhttp.WithServerRequestName(apmhttp.UnknownRouteRequestName),
		apmhttp.WithServerRequestIgnorer(opts.requestIgnorer),
	)
}

func gatherOptions(o ...Option) options {
	opts := options{
		tracer:         apm.DefaultTracer,
		requestIgnorer: apmhttp.DefaultServerRequestIgnorer(),
	}
	for _, o := range o {
		o(&opts)
	}
	if opts.recovery == nil {
		opts.recovery = apmhttp.NewTraceRecovery(opts.tracer)
	}
	return opts
}

type options struct {
	tracer         *apm.Tracer
	recovery       apmhttp.RecoveryFunc
	requestIgnorer apmhttp.RequestIgnorerFunc
}

// Option sets options for tracing.
type Option func(*options)

// WithTracer returns an Option which sets t as the tracer
// to use for tracing server requests.
func WithTracer(t *apm.Tracer) Option {
	if t == nil {
		panic("t == nil")
	}
	return func(o *options) {
		o.tracer = t
	}
}

// WithRecovery returns an Option which sets r as the recovery
// function to use for tracing server requests.
func WithRecovery(r apmhttp.RecoveryFunc) Option {
	if r == nil {
		panic("r == nil")
	}
	return func(o *options) {
		o.recovery = r
	}
}

// WithRequestIgnorer returns a Option which sets r as the
// function to use to determine whether or not a request should
// be ignored. If r is nil, all requests will be reported.
func WithRequestIgnorer(r apmhttp.RequestIgnorerFunc) Option {
	if r == nil {
		r = apmhttp.IgnoreNone
	}
	return func(o *options) {
		o.requestIgnorer = r
	}
}
