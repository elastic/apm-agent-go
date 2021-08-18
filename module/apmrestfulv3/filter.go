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

//go:build go1.11
// +build go1.11

package apmrestfulv3 // import "go.elastic.co/apm/module/apmrestfulv3"

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"

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
		tracer: apm.DefaultTracer,
	}
	for _, o := range o {
		o(&opts)
	}
	if opts.requestIgnorer == nil {
		opts.requestIgnorer = apmhttp.NewDynamicServerRequestIgnorer(opts.tracer)
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
	if !f.tracer.Recording() || f.requestIgnorer(req.Request) {
		chain.ProcessFilter(req, resp)
		return
	}

	var name string
	if routePath := massageRoutePath(req.SelectedRoutePath()); routePath != "" {
		name = req.Request.Method + " " + massageRoutePath(req.SelectedRoutePath())
	} else {
		name = apmhttp.UnknownRouteRequestName(req.Request)
	}
	tx, body, httpRequest := apmhttp.StartTransactionWithBody(f.tracer, name, req.Request)
	defer tx.End()
	req.Request = httpRequest

	const frameworkName = "go-restful"
	const frameworkVersion = "v3"
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
		body.Discard()
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
