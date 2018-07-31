package apmgorilla

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/module/apmhttp"
)

// Middleware returns a new gorilla/mux middleware handler
// for tracing requests and reporting errors.
//
// This middleware will recover and report panics, so it can
// be used instead of the gorilla/middleware.RecoveryHandler
// middleware.
//
// By default, the middleware will use elasticapm.DefaultTracer.
// Use WithTracer to specify an alternative tracer.
func Middleware(o ...Option) mux.MiddlewareFunc {
	opts := options{
		tracer:         elasticapm.DefaultTracer,
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
	route := mux.CurrentRoute(req)
	if route != nil {
		tpl, err := route.GetPathTemplate()
		if err == nil {
			return req.Method + " " + massageTemplate(tpl)
		}
	}
	return apmhttp.ServerRequestName(req)
}

type options struct {
	tracer         *elasticapm.Tracer
	requestIgnorer apmhttp.RequestIgnorerFunc
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
