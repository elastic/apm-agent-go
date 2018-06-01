package apmecho

import (
	"errors"
	"fmt"

	"github.com/labstack/echo"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/module/apmhttp"
)

// Middleware returns a new Echo middleware handler for tracing
// requests and reporting errors.
//
// This middleware will recover and report panics, so it can
// be used instead of echo/middleware.Recover.
//
// By default, the middleware will use elasticapm.DefaultTracer.
// Use WithTracer to specify an alternative tracer.
func Middleware(o ...Option) echo.MiddlewareFunc {
	opts := options{tracer: elasticapm.DefaultTracer}
	for _, o := range o {
		o(&opts)
	}
	return func(h echo.HandlerFunc) echo.HandlerFunc {
		m := &middleware{tracer: opts.tracer, handler: h}
		return m.handle
	}
}

type middleware struct {
	handler echo.HandlerFunc
	tracer  *elasticapm.Tracer
}

func (m *middleware) handle(c echo.Context) error {
	if !m.tracer.Active() {
		return m.handler(c)
	}
	req := c.Request()
	name := req.Method + " " + c.Path()
	tx := m.tracer.StartTransaction(name, "request")
	ctx := elasticapm.ContextWithTransaction(req.Context(), tx)
	req = apmhttp.RequestWithContext(ctx, req)
	c.SetRequest(req)
	defer tx.End()
	body := m.tracer.CaptureHTTPRequestBody(req)

	defer func() {
		if v := recover(); v != nil {
			e := m.tracer.Recovered(v, tx)
			e.Context.SetHTTPRequest(req)
			e.Context.SetHTTPRequestBody(body)
			err, ok := v.(error)
			if !ok {
				err = errors.New(fmt.Sprint(v))
			}
			e.Send()
			c.Error(err)
		}
	}()

	resp := c.Response()
	handlerErr := m.handler(c)
	tx.Result = apmhttp.StatusCodeResult(resp.Status)
	if tx.Sampled() {
		tx.Context.SetHTTPRequest(req)
		tx.Context.SetHTTPRequestBody(body)
		tx.Context.SetHTTPStatusCode(resp.Status)
		tx.Context.SetHTTPResponseHeaders(resp.Header())
		tx.Context.SetHTTPResponseHeadersSent(resp.Committed)
	}
	if handlerErr != nil {
		e := m.tracer.NewError(handlerErr)
		e.Context.SetHTTPRequest(req)
		e.Context.SetHTTPRequestBody(body)
		e.Transaction = tx
		e.Handled = true
		e.Send()
		return handlerErr
	}
	return nil
}

type options struct {
	tracer *elasticapm.Tracer
}

// Option sets options for tracing.
type Option func(*options)

// WithTracer returns an Option which sets t as the tracer
// to use for tracing server requests.
func WithTracer(t *elasticapm.Tracer) Option {
	if t == nil {
		panic("t == nil")
	}
	return func(o *options) {
		o.tracer = t
	}
}
