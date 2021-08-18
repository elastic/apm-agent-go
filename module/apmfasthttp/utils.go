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

//go:build go1.12
// +build go1.12

package apmfasthttp // import "go.elastic.co/apm/module/apmfasthttp"

import (
	"net/url"

	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

func getRequestTraceparent(ctx *fasthttp.RequestCtx, header string) (apm.TraceContext, bool) {
	if value := ctx.Request.Header.Peek(header); len(value) > 0 {
		if c, err := apmhttp.ParseTraceparentHeader(string(value)); err == nil {
			return c, true
		}
	}

	return apm.TraceContext{}, false
}

// NewDynamicServerRequestIgnorer returns the RequestIgnorer to use in
// handler. The list of wildcard patterns comes from central config
func NewDynamicServerRequestIgnorer(t *apm.Tracer) RequestIgnorerFunc {
	return func(ctx *fasthttp.RequestCtx) bool {
		uri := string(ctx.Request.URI().RequestURI())

		u, err := url.ParseRequestURI(uri)
		if err != nil {
			return true
		}

		return t.IgnoredTransactionURL(u)
	}
}

// ServerRequestName returns the transaction name for the server request context, ctx.
func ServerRequestName(ctx *fasthttp.RequestCtx) string {
	b := bytebufferpool.Get()
	defer bytebufferpool.Put(b)

	b.Write(ctx.Request.Header.Method())
	b.WriteByte(' ')
	b.Write(ctx.Request.URI().Path())

	return b.String()
}
