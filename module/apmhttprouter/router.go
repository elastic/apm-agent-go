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

package apmhttprouter // import "go.elastic.co/apm/module/apmhttprouter"

import (
	"context"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// Router wraps an httprouter.Router, instrumenting all added routes
// except static content served with ServeFiles.
type Router struct {
	*httprouter.Router
	opts []Option
}

// New returns a new Router which will instrument all added routes
// except static content served with ServeFiles.
//
// Router.NotFound and Router.MethodNotAllowed will be set, and will
// report transactions with the name "<METHOD> unknown route".
func New(o ...Option) *Router {
	router := httprouter.New()
	router.NotFound = WrapNotFoundHandler(router.NotFound, o...)
	router.MethodNotAllowed = WrapMethodNotAllowedHandler(router.MethodNotAllowed, o...)
	return &Router{
		Router: router,
		opts:   o,
	}
}

// DELETE calls r.Router.DELETE with a wrapped handler.
func (r *Router) DELETE(path string, handle httprouter.Handle) {
	r.Router.DELETE(path, Wrap(handle, path, r.opts...))
}

// GET calls r.Router.GET with a wrapped handler.
func (r *Router) GET(path string, handle httprouter.Handle) {
	r.Router.GET(path, Wrap(handle, path, r.opts...))
}

// HEAD calls r.Router.HEAD with a wrapped handler.
func (r *Router) HEAD(path string, handle httprouter.Handle) {
	r.Router.HEAD(path, Wrap(handle, path, r.opts...))
}

// Handle calls r.Router.Handle with a wrapped handler.
func (r *Router) Handle(method, path string, handle httprouter.Handle) {
	r.Router.Handle(method, path, Wrap(handle, path, r.opts...))
}

// HandlerFunc is equivalent to r.Router.HandlerFunc, but traces requests.
func (r *Router) HandlerFunc(method, path string, handler http.HandlerFunc) {
	r.Handler(method, path, handler)
}

// Handler is equivalent to r.Router.Handler, but traces requests.
func (r *Router) Handler(method, path string, handler http.Handler) {
	r.Handle(method, path, func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
		ctx := req.Context()
		ctx = context.WithValue(ctx, httprouter.ParamsKey, p)
		req = req.WithContext(ctx)
		handler.ServeHTTP(w, req)
	})
}

// OPTIONS is equivalent to r.Router.OPTIONS, but traces requests.
func (r *Router) OPTIONS(path string, handle httprouter.Handle) {
	r.Router.OPTIONS(path, Wrap(handle, path, r.opts...))
}

// PATCH is equivalent to r.Router.PATCH, but traces requests.
func (r *Router) PATCH(path string, handle httprouter.Handle) {
	r.Router.PATCH(path, Wrap(handle, path, r.opts...))
}

// POST is equivalent to r.Router.POST, but traces requests.
func (r *Router) POST(path string, handle httprouter.Handle) {
	r.Router.POST(path, Wrap(handle, path, r.opts...))
}

// PUT is equivalent to r.Router.PUT, but traces requests.
func (r *Router) PUT(path string, handle httprouter.Handle) {
	r.Router.PUT(path, Wrap(handle, path, r.opts...))
}
