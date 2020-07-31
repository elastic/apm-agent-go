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

package apmhttp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http/httptrace"

	"go.elastic.co/apm"
)

// WithClientTrace returns a ClientOption for
// tracing events within HTTP client requests.
func WithClientTrace() ClientOption {
	return func(rt *roundTripper) {
		rt.requestTracer = &requestTracer{}
	}
}

type requestTracer struct {
	tx *apm.Transaction

	DNS,
	Connect,
	TLS,
	Request,
	Response *apm.Span
}

func (r *requestTracer) start(ctx context.Context) context.Context {
	if r == nil {
		return ctx
	}
	r.tx = apm.TransactionFromContext(ctx)
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		DNSStart: func(i httptrace.DNSStartInfo) {
			r.DNS = r.tx.StartSpan(fmt.Sprintf("DNS %s", i.Host), "http.dns", nil)
		},

		DNSDone: func(i httptrace.DNSDoneInfo) {
			r.DNS.End()
		},

		ConnectStart: func(_, _ string) {
			r.Connect = r.tx.StartSpan("Connect", "http.connect", nil)

			if r.DNS == nil {
				r.Connect.Context.SetLabel("dns", false)
			}
		},

		ConnectDone: func(network, addr string, err error) {
			r.Connect.End()
		},

		TLSHandshakeStart: func() {
			r.TLS = r.tx.StartSpan("TLS", "http.tls", nil)
		},

		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			r.TLS.End()
		},

		WroteRequest: func(info httptrace.WroteRequestInfo) {
			r.Request = r.tx.StartSpan("Request", "http.request", nil)
		},

		GotFirstResponseByte: func() {
			r.Request.End()
			r.Response = r.tx.StartSpan("Response", "http.response", nil)
		},
	})
}

func (r *requestTracer) end() {
	if r == nil {
		return
	}
	if r.Response != nil {
		r.Response.End()
	}
}
