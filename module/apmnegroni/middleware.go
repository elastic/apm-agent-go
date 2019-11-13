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

package apmnegroni

import (
	"context"
	"net/http"

	"github.com/urfave/negroni"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/stacktrace"
)

func init() {
	stacktrace.RegisterLibraryPackage(
		"github.com/urfave/negroni",
	)
}

// Middleware returns a new negroni middleware handler for tracing
// requests and reporting errors.
//
// This middleware will recover and report panics, so it can
// be used instead of the standard negroni.Recovery middleware.
//
// By default, the middleware will use apm.DefaultTracer.
// Use WithTracer to specify an alternative tracer.
func Middleware(o ...Option) negroni.Handler {
	m := &middleware{
		handler: apmhttp.Wrap(http.HandlerFunc(nextHandler), apmhttpServerOptions(o...)...),
	}
	return m
}

type middleware struct {
	handler http.Handler
}

func (m *middleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	r = r.WithContext(context.WithValue(r.Context(), nextKey{}, next))
	m.handler.ServeHTTP(w, r)
}

type nextKey struct{}

func nextHandler(w http.ResponseWriter, r *http.Request) {
	next := r.Context().Value(nextKey{}).(http.HandlerFunc)
	next(w, r)
}

// Option sets options for tracing.
type Option apmhttp.ServerOption

func apmhttpServerOptions(o ...Option) []apmhttp.ServerOption {
	opts := make([]apmhttp.ServerOption, len(o))
	for i, opt := range o {
		opts[i] = apmhttp.ServerOption(opt)
	}
	return opts
}

// RecoveryFunc is the type of a function for use in WithRecovery.
type RecoveryFunc apmhttp.RecoveryFunc

// RequestNameFunc is the type of a function for use in
// WithServerRequestName.
type RequestNameFunc apmhttp.RequestNameFunc

// WithServerRequestName returns a Option which sets r as the function
// to use to obtain the transaction name for the given server request.
func WithServerRequestName(r RequestNameFunc) Option {
	return Option(apmhttp.WithServerRequestName(apmhttp.RequestNameFunc(r)))
}

// WithTracer returns a Option which sets t as the tracer
// to use for tracing server requests.
func WithTracer(t *apm.Tracer) Option {
	return Option(apmhttp.WithTracer(t))
}

// WithRecovery returns a Option which sets r as the recovery
// function to use for tracing server requests.
func WithRecovery(r RecoveryFunc) Option {
	return Option(apmhttp.WithRecovery(apmhttp.RecoveryFunc(r)))
}

// WithPanicPropagation returns a Option which enable panic propagation.
// Any panic will be recovered and recorded as an error in a transaction, then
// panic will be caused again.
func WithPanicPropagation() Option {
	return Option(apmhttp.WithPanicPropagation())
}

// RequestIgnorerFunc is the type of a function for use in
// WithServerRequestIgnorer.
type RequestIgnorerFunc apmhttp.RequestIgnorerFunc

// WithServerRequestIgnorer returns a Option which sets r as the
// function to use to determine whether or not a server request should
// be ignored. If r is nil, all requests will be reported.
func WithServerRequestIgnorer(r RequestIgnorerFunc) Option {
	return Option(apmhttp.WithServerRequestIgnorer(apmhttp.RequestIgnorerFunc(r)))
}
