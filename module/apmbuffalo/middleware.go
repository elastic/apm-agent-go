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

package apmbuffalo

import (
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/runtime"
	"github.com/pkg/errors"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

// Instrument instruments the buffalo.App so that requests are traced.
//
// Instrument installs middleware into r, and also overrides
// r.NotFoundHandler and r.MethodNotAllowedHandler so that they
// are traced. If you modify either of those fields, you must do so
// before calling Instrument.
func Instrument(r *buffalo.App, o ...Option) {
	m := Middleware(o...)
	r.Use(m)
	r.Muxer().NotFoundHandler = WrapNotFoundHandler(r.Muxer().NotFoundHandler)
	r.Muxer().MethodNotAllowedHandler = WrapMethodNotAllowedHandler(r.Muxer().MethodNotAllowedHandler)
}

type options struct {
	tracer         *apm.Tracer
	recovery       apmhttp.RecoveryFunc
	requestIgnorer apmhttp.RequestIgnorerFunc
}

// Option sets options for tracing.
type Option func(*options)

// Middleware returns a new buffalo.MiddlewareFunc middleware handler for tracing
// requests and reporting errors.
//
// This middleware will recover and report panics and propagate the panic
// value to buffalo.PanicHandler
//
// By default, the middleware will use apm.DefaultTracer.
// Use WithTracer to specify an alternative tracer.
func Middleware(o ...Option) buffalo.MiddlewareFunc {
	opts := options{
		tracer:         apm.DefaultTracer,
		requestIgnorer: apmhttp.DefaultServerRequestIgnorer(),
	}
	for _, o := range o {
		o(&opts)
	}

	return func(h buffalo.Handler) buffalo.Handler {
		m := &middleware{
			tracer:         opts.tracer,
			handler:        h,
			requestIgnorer: opts.requestIgnorer,
		}
		return m.handle
	}
}

type middleware struct {
	handler        buffalo.Handler
	tracer         *apm.Tracer
	requestIgnorer apmhttp.RequestIgnorerFunc
}

func (m *middleware) handle(c buffalo.Context) error {
	req := c.Request()
	if !m.tracer.Active() || m.requestIgnorer(req) {
		return m.handler(c)
	}
	var requestName string
	if routeInfo, ok := c.Value("current_route").(buffalo.RouteInfo); ok {
		requestName = req.Method + " " + massageTemplate(routeInfo.Path)
	} else {
		requestName = apmhttp.UnknownRouteRequestName(req)
	}

	tx, req := apmhttp.StartTransaction(m.tracer, requestName, req)
	defer tx.End()
	body := m.tracer.CaptureHTTPRequestBody(req)
	resp := c.Response()
	var handlerErr error
	defer func() {
		var r *buffalo.Response
		if v := recover(); v != nil {
			e := m.tracer.Recovered(v)
			e.SetTransaction(tx)
			r, _ := resp.(*buffalo.Response)
			setContext(&e.Context, req, body, http.StatusInternalServerError, r.Header())
			e.Send()
			panic(v) // propagate panic to top panic handler
		}
		r, _ = resp.(*buffalo.Response)
		tx.Result = apmhttp.StatusCodeResult(r.Status)
		if tx.Sampled() {
			setContext(&tx.Context, req, body, r.Status, r.Header())
		}
	}()
	handlerErr = m.handler(c)
	if handlerErr != nil {
		cause := errors.Cause(handlerErr)
		var status int
		if httpErr, ok := cause.(buffalo.HTTPError); ok {
			status = httpErr.Status
		} else {
			status = http.StatusInternalServerError
		}
		r, _ := resp.(*buffalo.Response)
		e := m.tracer.NewError(handlerErr)
		setContext(&e.Context, req, body, status, r.Header())
		e.SetTransaction(tx)
		e.Handled = true
		e.Send()
	}
	return handlerErr
}

func setContext(ctx *apm.Context, req *http.Request, body *apm.BodyCapturer, status int, headers http.Header) {
	ctx.SetFramework("gobuffalo", runtime.Version)
	ctx.SetHTTPRequest(req)
	ctx.SetHTTPRequestBody(body)
	ctx.SetHTTPStatusCode(status)
	ctx.SetHTTPResponseHeaders(headers)
}

// WrapNotFoundHandler wraps h with m. If h is nil, then http.NotFoundHandler() will be used.
func WrapNotFoundHandler(h http.Handler) http.Handler {
	if h == nil {
		h = http.NotFoundHandler()
	}
	return h
}

// WrapMethodNotAllowedHandler wraps h with m. If h is nil, then a default handler
// will be used that returns status code 405.
func WrapMethodNotAllowedHandler(h http.Handler) http.Handler {
	if h == nil {
		h = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusMethodNotAllowed)
		})
	}
	return h
}

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
