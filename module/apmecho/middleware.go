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

package apmecho

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

// Middleware returns a new Echo middleware handler for tracing
// requests and reporting errors.
//
// This middleware will recover and report panics, so it can
// be used instead of echo/middleware.Recover.
//
// By default, the middleware will use apm.DefaultTracer.
// Use WithTracer to specify an alternative tracer.
func Middleware(o ...Option) echo.MiddlewareFunc {
	opts := options{
		tracer:         apm.DefaultTracer,
		requestIgnorer: apmhttp.DefaultServerRequestIgnorer(),
	}
	for _, o := range o {
		o(&opts)
	}
	return func(h echo.HandlerFunc) echo.HandlerFunc {
		m := &middleware{
			tracer:         opts.tracer,
			handler:        h,
			requestIgnorer: opts.requestIgnorer,
		}
		return m.handle
	}
}

type middleware struct {
	handler        echo.HandlerFunc
	tracer         *apm.Tracer
	requestIgnorer apmhttp.RequestIgnorerFunc
}

func (m *middleware) handle(c echo.Context) error {
	req := c.Request()
	if !m.tracer.Active() || m.requestIgnorer(req) {
		return m.handler(c)
	}
	name := req.Method + " " + c.Path()
	tx, req := apmhttp.StartTransaction(m.tracer, name, req)
	defer tx.End()
	c.SetRequest(req)
	body := m.tracer.CaptureHTTPRequestBody(req)

	resp := c.Response()
	var handlerErr error
	defer func() {
		if v := recover(); v != nil {
			err, ok := v.(error)
			if !ok {
				err = errors.New(fmt.Sprint(v))
			}
			c.Error(err)

			e := m.tracer.Recovered(v)
			e.SetTransaction(tx)
			setContext(&e.Context, req, resp, body)
			e.Send()
		}
		if handlerErr != nil {
			e := m.tracer.NewError(handlerErr)
			setContext(&e.Context, req, resp, body)
			e.SetTransaction(tx)
			e.Handled = true
			e.Send()
		}
		tx.Result = apmhttp.StatusCodeResult(resp.Status)
		if tx.Sampled() {
			setContext(&tx.Context, req, resp, body)
		}
	}()

	handlerErr = m.handler(c)
	if handlerErr == nil && !resp.Committed {
		resp.WriteHeader(http.StatusOK)
	}
	return handlerErr
}

func setContext(ctx *apm.Context, req *http.Request, resp *echo.Response, body *apm.BodyCapturer) {
	ctx.SetFramework("echo", echo.Version)
	ctx.SetHTTPRequest(req)
	ctx.SetHTTPRequestBody(body)
	ctx.SetHTTPStatusCode(resp.Status)
	ctx.SetHTTPResponseHeaders(resp.Header())
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
