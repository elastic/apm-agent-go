package apmecho

import (
	"errors"
	"fmt"

	"github.com/labstack/echo"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/contrib/apmhttp"
)

// Middleware returns a new Echo middleware handler for tracing
// requests and reporting errors, using the given tracer, or
// elasticapm.DefaultTracer if the tracer is nil.
//
// This middleware will recover and report panics, so it can
// be used instead of echo/middleware.Recover.
func Middleware(tracer *elasticapm.Tracer) echo.MiddlewareFunc {
	if tracer == nil {
		tracer = elasticapm.DefaultTracer
	}
	return func(h echo.HandlerFunc) echo.HandlerFunc {
		m := &middleware{tracer: tracer, handler: h}
		return m.handle
	}
}

type middleware struct {
	handler echo.HandlerFunc
	tracer  *elasticapm.Tracer
}

func (m *middleware) handle(c echo.Context) error {
	req := c.Request()
	name := req.Method + " " + c.Path()
	tx := m.tracer.StartTransaction(name, "request")
	if tx.Ignored() {
		tx.Discard()
		return m.handler(c)
	}

	ctx := elasticapm.ContextWithTransaction(req.Context(), tx)
	req = req.WithContext(ctx)
	c.SetRequest(req)
	defer tx.Done(-1)
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
