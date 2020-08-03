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
		rt.traceRequests = true
	}
}

type requestTracer struct {
	tx *apm.Transaction

	DNS,
	TLS,
	Request,
	Response *apm.Span

	Connects map[[2]string]*apm.Span
}

func connectKey(network, addr string) [2]string {
	return [2]string{network, addr}
}

func (r *requestTracer) start(ctx context.Context, parent *apm.Span) context.Context {
	r.tx = apm.TransactionFromContext(ctx)
	r.Connects = make(map[[2]string]*apm.Span)
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		DNSStart: func(i httptrace.DNSStartInfo) {
			r.DNS = r.tx.StartSpan(fmt.Sprintf("DNS %s", i.Host), "http.dns", parent)
		},

		DNSDone: func(i httptrace.DNSDoneInfo) {
			r.DNS.End()
		},

		ConnectStart: func(network, addr string) {
			span := r.tx.StartSpan(fmt.Sprintf("Connect %s", addr), "http.connect", parent)
			if r.DNS == nil {
				span.Context.SetLabel("dns", false)
			}
			r.Connects[connectKey(network, addr)] = span
		},

		ConnectDone: func(network, addr string, err error) {
			span := r.Connects[connectKey(network, addr)]
			if err != nil {
				span.Context.SetLabel("error", err.Error())
			}
			span.End()
		},

		TLSHandshakeStart: func() {
			r.TLS = r.tx.StartSpan("TLS", "http.tls", parent)
		},

		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			r.TLS.End()
		},

		WroteRequest: func(info httptrace.WroteRequestInfo) {
			r.Request = r.tx.StartSpan("Request", "http.request", parent)
		},

		GotFirstResponseByte: func() {
			r.Request.End()
			r.Response = r.tx.StartSpan("Response", "http.response", parent)
		},
	})
}

func (r *requestTracer) end() {
	if r.Response != nil {
		r.Response.End()
	}
}
