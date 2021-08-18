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

//go:build go1.9
// +build go1.9

package apmgrpc // import "go.elastic.co/apm/module/apmgrpc"

import (
	"net"
	"net/http"
	"net/url"
	"sync"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

// NewUnaryClientInterceptor returns a grpc.UnaryClientInterceptor that
// traces gRPC requests with the given options.
//
// The interceptor will trace spans with the "external.grpc" type for each
// request made, for any client method presented with a context containing
// a sampled apm.Transaction.
func NewUnaryClientInterceptor(o ...ClientOption) grpc.UnaryClientInterceptor {
	opts := clientOptions{}
	for _, o := range o {
		o(&opts)
	}
	return func(
		ctx context.Context,
		method string,
		req, resp interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		var peer peer.Peer     // maybe set after call if span != nil
		var header metadata.MD // maybe set after call if span != nil
		span, ctx := startSpan(ctx, method)
		if span != nil {
			defer span.End()
			opts = append(opts, grpc.Peer(&peer), grpc.Header(&header))
		}
		err := invoker(ctx, method, req, resp, cc, opts...)
		if span != nil {
			setSpanOutcome(span, err)
			url := url.URL{Scheme: "http", Path: method}
			if _, ok := peer.AuthInfo.(credentials.TLSInfo); ok {
				url.Scheme = "https"
			}
			if peer.Addr != nil {
				url.Host = peer.Addr.String()
			}
			span.Context.SetHTTPRequest(&http.Request{
				URL:        &url,
				Method:     "POST", // method is always POST
				ProtoMajor: 2,
				ProtoMinor: 0,
				Header:     http.Header(header),
			})
			if url.Host != "" {
				span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
					Name:     url.Host,
					Resource: url.Host,
				})
			}
		}
		return err
	}
}

// NewStreamClientInterceptor returns a grpc.UnaryClientInterceptor that
// traces gRPC requests with the given options.
//
// The interceptor will trace spans with the "external.grpc" type for each
// stream request made, for any client method presented with a context
// containing a sampled apm.Transaction.
//
// Spans are ended when the stream is closed, which can happen in various
// ways: the initial stream setup request fails, Header, SendMsg or RecvMsg
// return with an error, or RecvMsg returns for a non-streaming server.
func NewStreamClientInterceptor(o ...ClientOption) grpc.StreamClientInterceptor {
	opts := clientOptions{}
	for _, o := range o {
		o(&opts)
	}
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		var peer peer.Peer
		span, ctx := startSpan(ctx, method)
		if span != nil {
			opts = append(opts, grpc.Peer(&peer))
		}
		stream, err := streamer(ctx, desc, cc, method, opts...)
		if span != nil {
			if err != nil {
				setSpanOutcome(span, err)
				setSpanContext(span, peer)
				span.End()
			} else if stream != nil {
				wrapped := &clientStream{ClientStream: stream}
				go func() {
					defer span.End()
					// Header blocks until headers are available
					// or the stream is ended. Either way, after
					// Header returns, it is safe to call Context().
					stream.Header()
					<-stream.Context().Done()
					err := wrapped.getError()
					setSpanOutcome(span, err)
					setSpanContext(span, peer)
				}()
				stream = wrapped
			}
		}
		return stream, err
	}
}

// clientStream wraps grpc.ClientStream to intercept errors.
type clientStream struct {
	grpc.ClientStream
	mu  sync.RWMutex
	err error
}

func (s *clientStream) CloseSend() error {
	err := s.ClientStream.CloseSend()
	s.setError(err)
	return err
}

func (s *clientStream) Header() (metadata.MD, error) {
	md, err := s.ClientStream.Header()
	s.setError(err)
	return md, err
}

func (s *clientStream) SendMsg(m interface{}) error {
	err := s.ClientStream.SendMsg(m)
	s.setError(err)
	return err
}

func (s *clientStream) RecvMsg(m interface{}) error {
	err := s.ClientStream.RecvMsg(m)
	s.setError(err)
	return err
}

func (s *clientStream) getError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.err
}

func (s *clientStream) setError(err error) {
	if err != nil {
		s.mu.Lock()
		s.err = err
		s.mu.Unlock()
	}
}

func startSpan(ctx context.Context, name string) (*apm.Span, context.Context) {
	tx := apm.TransactionFromContext(ctx)
	if tx == nil {
		return nil, ctx
	}
	traceContext := tx.TraceContext()
	propagateLegacyHeader := tx.ShouldPropagateLegacyHeader()
	if !traceContext.Options.Recorded() {
		return nil, outgoingContextWithTraceContext(ctx, traceContext, propagateLegacyHeader)
	}
	span := tx.StartSpan(name, "external.grpc", apm.SpanFromContext(ctx))
	if !span.Dropped() {
		traceContext = span.TraceContext()
		ctx = apm.ContextWithSpan(ctx, span)
	}
	return span, outgoingContextWithTraceContext(ctx, traceContext, propagateLegacyHeader)
}

func setSpanContext(span *apm.Span, peer peer.Peer) {
	if peer.Addr != nil {
		if tcpAddr, ok := peer.Addr.(*net.TCPAddr); ok {
			span.Context.SetDestinationAddress(tcpAddr.IP.String(), tcpAddr.Port)
		}
		addrString := peer.Addr.String()
		span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
			Name:     addrString,
			Resource: addrString,
		})
	}
}

func outgoingContextWithTraceContext(
	ctx context.Context,
	traceContext apm.TraceContext,
	propagateLegacyHeader bool,
) context.Context {
	traceparentValue := apmhttp.FormatTraceparentHeader(traceContext)
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.Pairs(w3cTraceparentHeader, traceparentValue)
	} else {
		md = md.Copy()
		md.Set(w3cTraceparentHeader, traceparentValue)
	}
	if propagateLegacyHeader {
		md.Set(elasticTraceparentHeader, traceparentValue)
	}
	if tracestate := traceContext.State.String(); tracestate != "" {
		md.Set(tracestateHeader, tracestate)
	}
	return metadata.NewOutgoingContext(ctx, md)
}

func setSpanOutcome(span *apm.Span, err error) {
	statusCode := statusCodeFromError(err)

	// On the client side, all codes except for OK are treated as failures
	// by default, and can be overridden by setting the Outcome explicitly.
	if span.Outcome == "" {
		switch statusCode {
		case codes.OK:
			span.Outcome = "success"
		default:
			span.Outcome = "failure"
		}
	}
}

type clientOptions struct {
	tracer *apm.Tracer
}

// ClientOption sets options for client-side tracing.
type ClientOption func(*clientOptions)
