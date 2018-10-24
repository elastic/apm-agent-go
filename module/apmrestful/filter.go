package apmrestful

import (
	"net/http"

	"github.com/emicklei/go-restful"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

// Filter returns a new restful.Filter for tracing requests
// and recovering and reporting panics to Elastic APM.
//
// By default, the filter will use apm.DefaultTracer.
// Use WithTracer to specify an alternative tracer.
func Filter(o ...Option) restful.FilterFunction {
	opts := options{
		tracer:         apm.DefaultTracer,
		requestIgnorer: apmhttp.DefaultServerRequestIgnorer(),
	}
	for _, o := range o {
		o(&opts)
	}
	return (&filter{
		tracer:         opts.tracer,
		requestIgnorer: opts.requestIgnorer,
	}).filter
}

type filter struct {
	tracer         *apm.Tracer
	requestIgnorer apmhttp.RequestIgnorerFunc
}

func (f *filter) filter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	if !f.tracer.Active() || f.requestIgnorer(req.Request) {
		chain.ProcessFilter(req, resp)
		return
	}

	name := req.Request.Method + " " + massageRoutePath(req.SelectedRoutePath())
	tx, httpRequest := apmhttp.StartTransaction(f.tracer, name, req.Request)
	defer tx.End()
	req.Request = httpRequest
	body := f.tracer.CaptureHTTPRequestBody(httpRequest)

	const frameworkName = "go-restful"
	const frameworkVersion = ""
	if tx.Sampled() {
		tx.Context.SetFramework(frameworkName, frameworkVersion)
	}

	origResponseWriter := resp.ResponseWriter
	w, httpResp := apmhttp.WrapResponseWriter(origResponseWriter)
	resp.ResponseWriter = w
	defer func() {
		resp.ResponseWriter = origResponseWriter
		if v := recover(); v != nil {
			if httpResp.StatusCode == 0 {
				w.WriteHeader(http.StatusInternalServerError)
			}
			e := f.tracer.Recovered(v)
			e.SetTransaction(tx)
			apmhttp.SetContext(&e.Context, req.Request, httpResp, body)
			e.Context.SetFramework(frameworkName, frameworkVersion)
			e.Send()
		}
		apmhttp.SetTransactionContext(tx, req.Request, httpResp, body)
	}()
	chain.ProcessFilter(req, resp)
	if httpResp.StatusCode == 0 {
		httpResp.StatusCode = http.StatusOK
	}
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
