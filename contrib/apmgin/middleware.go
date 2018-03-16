package apmgin

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/contrib/apmhttp"
	"github.com/elastic/apm-agent-go/model"
)

// Framework is a model.Framework initialized with values
// describing the gin framework name and version.
var Framework = model.Framework{
	Name:    "gin",
	Version: gin.Version,
}

func init() {
	if elasticapm.DefaultTracer.Service.Framework == nil {
		// TODO(axw) this is not ideal, as there could be multiple
		// frameworks in use within a program. The intake API should
		// be extended to support specifying a framework on a
		// transaction, or perhaps specifying multiple frameworks
		// in the payload and referencing one from the transaction.
		elasticapm.DefaultTracer.Service.Framework = &Framework
	}
}

// Middleware returns a new Gin middleware handler for tracing
// requests and reporting errors, using the given tracer, or
// elasticapm.DefaultTracer if the tracer is nil.
//
// This middleware will recover and report panics, so it can
// be used instead of the standard gin.Recovery middleware.
func Middleware(engine *gin.Engine, tracer *elasticapm.Tracer) gin.HandlerFunc {
	if tracer == nil {
		tracer = elasticapm.DefaultTracer
	}
	m := &middleware{engine: engine, tracer: tracer}
	return m.handle
}

type middleware struct {
	engine *gin.Engine
	tracer *elasticapm.Tracer

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
	ctx := elasticapm.ContextWithTransaction(c.Request.Context(), tx)
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
