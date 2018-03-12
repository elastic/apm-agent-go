package apmgin

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elastic/apm-agent-go/contrib/apmhttp"
	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/trace"
)

// Framework is a model.Framework initialized with values
// describing the gin framework name and version.
var Framework = model.Framework{
	Name:    "gin",
	Version: gin.Version,
}

// Middleware returns a new Gin middleware handler for tracing
// requests and reporting errors.
//
// This middleware will recover and report panics, so it can
// be used instead of the standard gin.Recovery middleware.
func Middleware(engine *gin.Engine, tracer *trace.Tracer) gin.HandlerFunc {
	m := &middleware{engine: engine, tracer: tracer}
	return m.handle
}

type middleware struct {
	engine *gin.Engine
	tracer *trace.Tracer

	setRouteMapOnce sync.Once
	routeMap        map[string]map[string]string
}

func (m *middleware) handle(c *gin.Context) {
	m.setRouteMapOnce.Do(func() {
		routes := m.engine.Routes()
		rm := make(map[string]map[string]string)
		for _, r := range routes {
			mm := rm[r.Method]
			if mm == nil {
				mm = make(map[string]string)
				rm[r.Method] = mm
			}
			mm[r.Handler] = r.Path
		}
		m.routeMap = rm
	})

	requestName := c.Request.Method
	handlerName := c.HandlerName()
	if routePath, ok := m.routeMap[c.Request.Method][handlerName]; ok {
		requestName += " " + routePath
	}
	tx := m.tracer.StartTransaction(requestName, "request")
	ctx := trace.ContextWithTransaction(c.Request.Context(), tx)
	span := tx.StartSpan(requestName, tx.Type, nil)
	if span != nil {
		// TODO(axw) configurable span stack traces, off by default.
		// span.SetStacktrace(1)
		ctx = trace.ContextWithSpan(ctx, span)
	}
	c.Request = c.Request.WithContext(ctx)

	defer func() {
		duration := time.Since(tx.Timestamp)
		if v := recover(); v != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			e := m.tracer.Recovered(v, tx)
			if e.Exception.Stacktrace == nil {
				e.SetExceptionStacktrace(1)
			}
			e.Context = apmhttp.RequestContext(c.Request)
			e.Send()
		}
		tx.Result = strconv.Itoa(c.Writer.Status())
		var txContext *model.Context
		if tx.Sampled() || len(c.Errors) > 0 {
			// TODO(axw) optimize allocations below.
			finished := !c.IsAborted()
			written := c.Writer.Written()
			txContext = apmhttp.RequestContext(c.Request)
			txContext.Response = &model.Response{
				StatusCode:  c.Writer.Status(),
				Headers:     apmhttp.ResponseHeaders(c.Writer),
				HeadersSent: &written,
				Finished:    &finished,
			}
			txContext.Custom = map[string]interface{}{
				"gin": map[string]interface{}{
					"handler": handlerName,
				},
			}
		}
		if tx.Sampled() {
			tx.Context = txContext
		}
		if span != nil {
			span.Done(duration)
		}
		tx.Done(duration)

		// Report errors handled.
		for _, err := range c.Errors {
			e := m.tracer.NewError()
			e.Transaction = tx
			e.Context = txContext
			e.SetException(err.Err)
			e.Exception.Handled = true
			e.Send()
		}
	}()
	c.Next()
}
