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
	Server,
	Transfer *apm.Span
}

func (r *requestTracer) start(ctx context.Context) context.Context {
	if r == nil {
		return ctx
	}
	r.tx = apm.TransactionFromContext(ctx)
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		DNSStart: func(i httptrace.DNSStartInfo) {
			r.DNS = r.tx.StartSpan("DNS Lookup", "http.dns", nil)
		},

		DNSDone: func(i httptrace.DNSDoneInfo) {
			r.DNS.End()
		},

		ConnectStart: func(_, _ string) {
			r.Connect = r.tx.StartSpan("Connect", "http.connect", nil)

			if r.DNS == nil {
				r.tx.Context.SetLabel("dns", false)
			}
		},

		ConnectDone: func(network, addr string, err error) {
			r.Connect.End()
		},

		TLSHandshakeStart: func() {
			r.TLS = r.tx.StartSpan("TLS Handshake", "http.tls", nil)
			r.tx.Context.SetLabel("tls", true)
		},

		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			r.TLS.End()
		},

		PutIdleConn: func(err error) {
			r.tx.Context.SetLabel("conn_returned", err == nil)
		},

		GotConn: func(i httptrace.GotConnInfo) {
			// Handle when keep alive is used and connection is reused.
			// DNSStart(Done) and ConnectStart(Done) is skipped
			r.tx.Context.SetLabel("conn_reused", true)
		},

		WroteRequest: func(info httptrace.WroteRequestInfo) {
			r.Server = r.tx.StartSpan("Server Processing", "http.server", nil)

			// When connection is re-used, DNS/TCP/TLS hooks not called.
			if r.Connect == nil {
				// TODO
			}
		},

		GotFirstResponseByte: func() {
			r.Server.End()
			r.Transfer = r.tx.StartSpan("Transfer", "http.transfer", nil)
		},
	})
}

func (r *requestTracer) end() {
	if r == nil {
		return
	}
	if r.Transfer != nil {
		r.Transfer.End()
	}
}
