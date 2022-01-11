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

//go:build go1.13
// +build go1.13

package apmfiber // import "go.elastic.co/apm/module/apmfiber"

import (
	"net/http"

	"github.com/gofiber/fiber/v2"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmfasthttp"
	"go.elastic.co/apm/module/apmhttp"
)

// Middleware returns a new Fiber middleware handler for tracing
// requests and reporting errors.
//
// This middleware will recover and report panics, so it can
// be used instead of default recover middleware.
//
// By default, the middleware will use apm.DefaultTracer().
// Use WithTracer to specify an alternative tracer.
// Use WithPanicPropagation to disable panic recover.
func Middleware(o ...Option) fiber.Handler {
	m := &middleware{
		tracer:           apm.DefaultTracer(),
		requestIgnorer:   apmfasthttp.NewDynamicServerRequestIgnorer(apm.DefaultTracer()),
		panicPropagation: false,
	}

	for _, o := range o {
		o(m)
	}

	return m.handle
}

type middleware struct {
	tracer           *apm.Tracer
	requestIgnorer   apmfasthttp.RequestIgnorerFunc
	panicPropagation bool
}

func (m *middleware) handle(c *fiber.Ctx) error {
	reqCtx := c.Context()
	if !m.tracer.Recording() || m.requestIgnorer(reqCtx) {
		return c.Next()
	}

	name := string(reqCtx.Method()) + " " + c.Path()
	tx, body, err := apmfasthttp.StartTransactionWithBody(reqCtx, m.tracer, name)
	if err != nil {
		reqCtx.Error(err.Error(), http.StatusInternalServerError)

		return err
	}

	defer func() {
		resp := c.Response()
		path := c.Route().Path
		if path == "/" && resp.StatusCode() == http.StatusNotFound {
			tx.Name = string(reqCtx.Method()) + " unknown route"
		} else {
			// Workaround for set tx.Name as template path, not absolute
			tx.Name = string(reqCtx.Method()) + " " + path
		}

		if v := recover(); v != nil {
			if m.panicPropagation {
				defer panic(v)
			}

			e := m.tracer.Recovered(v)
			e.SetTransaction(tx)
			setContext(&e.Context, resp)
			e.Send()

			c.Status(http.StatusInternalServerError)
		}

		statusCode := resp.StatusCode()

		tx.Result = apmhttp.StatusCodeResult(statusCode)
		if tx.Sampled() {
			setContext(&tx.Context, resp)
		}

		body.Discard()
	}()

	nextErr := c.Next()
	if nextErr != nil {
		resp := c.Response()
		e := m.tracer.NewError(nextErr)
		e.Handled = true
		e.SetTransaction(tx)
		setContext(&e.Context, resp)
		e.Send()
	}

	return nextErr
}

func setContext(ctx *apm.Context, resp *fiber.Response) {
	ctx.SetFramework("fiber", fiber.Version)
	ctx.SetHTTPStatusCode(resp.StatusCode())

	headers := make(http.Header)
	resp.Header.VisitAll(func(k, v []byte) {
		sk := string(k)
		sv := string(v)

		headers.Set(sk, sv)
	})

	ctx.SetHTTPResponseHeaders(headers)
}

// Option sets options for tracing.
type Option func(*middleware)

// WithPanicPropagation returns an Option which enable panic propagation.
// Any panic will be recovered and recorded as an error in a transaction, then
// panic will be caused again.
func WithPanicPropagation() Option {
	return func(o *middleware) {
		o.panicPropagation = true
	}
}

// WithTracer returns an Option which sets t as the tracer
// to use for tracing server requests. If t is nil, using default tracer.
func WithTracer(t *apm.Tracer) Option {
	if t == nil {
		return noopOption
	}

	return func(o *middleware) {
		o.tracer = t
	}
}

// WithRequestIgnorer returns an Option which sets fn as the
// function to use to determine whether or not a request should
// be ignored. If fn is nil, using default RequestIgnorerFunc.
func WithRequestIgnorer(fn apmfasthttp.RequestIgnorerFunc) Option {
	if fn == nil {
		return noopOption
	}

	return func(o *middleware) {
		o.requestIgnorer = fn
	}
}

func noopOption(_ *middleware) {}
