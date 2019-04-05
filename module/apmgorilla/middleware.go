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

package apmgorilla

import (
	"net/http"

	"github.com/gorilla/mux"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

// Instrument instruments the mux.Router so that requests are traced.
//
// Instrument installs middleware into r, and alsos overrides
// r.NotFoundHandler and r.MethodNotAllowedHandler so that they
// are traced. If you modify either of those fields, you must do so
// before calling Instrument.
func Instrument(r *mux.Router, o ...Option) {
	m := Middleware(o...)
	r.Use(m)
	r.NotFoundHandler = WrapNotFoundHandler(r.NotFoundHandler, m)
	r.MethodNotAllowedHandler = WrapMethodNotAllowedHandler(r.MethodNotAllowedHandler, m)
}

// WrapNotFoundHandler wraps h with m. If h is nil, then http.NotFoundHandler() will be used.
func WrapNotFoundHandler(h http.Handler, m mux.MiddlewareFunc) http.Handler {
	if h == nil {
		h = http.NotFoundHandler()
	}
	return m(h)
}

// WrapMethodNotAllowedHandler wraps h with m. If h is nil, then a default handler
// will be used that returns status code 405.
func WrapMethodNotAllowedHandler(h http.Handler, m mux.MiddlewareFunc) http.Handler {
	if h == nil {
		h = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusMethodNotAllowed)
		})
	}
	return m(h)
}

// Middleware returns a new gorilla/mux middleware handler
// for tracing requests and reporting errors.
//
// This middleware will recover and report panics, so it can
// be used instead of the gorilla/middleware.RecoveryHandler
// middleware.
//
// Middleware does not get invoked when a route cannot be
// matched, or when an unsupported method is used. To report
// transactions in these cases, you should use the Instrument
// function, or set the router's NotFoundHandler and
// MethodNotAllowedHandler fields using the Wrap functions in
// this package.
//
// By default, the middleware will use apm.DefaultTracer.
// Use WithTracer to specify an alternative tracer.
func Middleware(o ...Option) mux.MiddlewareFunc {
	opts := options{
		tracer:         apm.DefaultTracer,
		requestIgnorer: apmhttp.DefaultServerRequestIgnorer(),
	}
	for _, o := range o {
		o(&opts)
	}
	return func(h http.Handler) http.Handler {
		return apmhttp.Wrap(
			h,
			apmhttp.WithTracer(opts.tracer),
			apmhttp.WithServerRequestName(routeRequestName),
			apmhttp.WithServerRequestIgnorer(opts.requestIgnorer),
		)
	}
}

func routeRequestName(req *http.Request) string {
	if route := mux.CurrentRoute(req); route != nil {
		tpl, err := route.GetPathTemplate()
		if err == nil {
			return req.Method + " " + massageTemplate(tpl)
		}
	}
	return apmhttp.UnknownRouteRequestName(req)
}

type options struct {
	tracer         *apm.Tracer
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
