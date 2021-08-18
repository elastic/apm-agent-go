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
	"crypto/tls"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

var (
	elasticTraceparentHeader = strings.ToLower(apmhttp.ElasticTraceparentHeader)
	w3cTraceparentHeader     = strings.ToLower(apmhttp.W3CTraceparentHeader)
	tracestateHeader         = strings.ToLower(apmhttp.TracestateHeader)
)

// NewUnaryServerInterceptor returns a grpc.UnaryServerInterceptor that
// traces gRPC requests with the given options.
//
// The interceptor will trace transactions with the "request" type for
// each incoming request. The transaction will be added to the context,
// so server methods can use apm.StartSpan with the provided context.
//
// By default, the interceptor will trace with apm.DefaultTracer,
// and will not recover any panics. Use WithTracer to specify an
// alternative tracer, and WithRecovery to enable panic recovery.
func NewUnaryServerInterceptor(o ...ServerOption) grpc.UnaryServerInterceptor {
	opts := serverOptions{
		tracer:         apm.DefaultTracer,
		recover:        false,
		requestIgnorer: DefaultServerRequestIgnorer(),
		streamIgnorer:  DefaultServerStreamIgnorer(),
	}
	for _, o := range o {
		o(&opts)
	}
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		if !opts.tracer.Recording() || opts.requestIgnorer(info) {
			return handler(ctx, req)
		}
		tx, ctx := startTransaction(ctx, opts.tracer, info.FullMethod)
		defer tx.End()

		// TODO(axw) define span context schema for RPC,
		// including at least the peer address.

		defer func() {
			r := recover()
			if r != nil {
				e := opts.tracer.Recovered(r)
				e.SetTransaction(tx)
				e.Context.SetFramework("grpc", grpc.Version)
				e.Handled = opts.recover
				e.Send()
				if opts.recover {
					err = status.Errorf(codes.Internal, "%s", r)
				} else {
					panic(r)
				}
			}
			setTransactionResult(tx, err)
		}()

		resp, err = handler(ctx, req)
		return resp, err
	}
}

// NewStreamServerInterceptor returns a grpc.StreamServerInterceptor that
// traces gRPC stream requests with the given options.
//
// The interceptor will trace transactions with the "request" type for each
// incoming stream request. The transaction will be added to the context, so
// server methods can use apm.StartSpan with the provided context.
//
// By default, the interceptor will trace with apm.DefaultTracer, and will
// not recover any panics. Use WithTracer to specify an alternative tracer,
// and WithRecovery to enable panic recovery.
func NewStreamServerInterceptor(o ...ServerOption) grpc.StreamServerInterceptor {
	opts := serverOptions{
		tracer:        apm.DefaultTracer,
		recover:       false,
		streamIgnorer: DefaultServerStreamIgnorer(),
	}
	for _, o := range o {
		o(&opts)
	}

	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		if !opts.tracer.Recording() || opts.streamIgnorer(info) {
			return handler(srv, stream)
		}
		ctx := stream.Context()
		tx, ctx := startTransaction(ctx, opts.tracer, info.FullMethod)
		defer tx.End()

		// TODO(axw) define span context schema for RPC,
		// including at least the peer address.

		defer func() {
			r := recover()
			if r != nil {
				e := opts.tracer.Recovered(r)
				e.SetTransaction(tx)
				e.Context.SetFramework("grpc", grpc.Version)
				e.Handled = opts.recover
				e.Send()
				if opts.recover {
					err = status.Errorf(codes.Internal, "%s", r)
				} else {
					panic(r)
				}
			}
			setTransactionResult(tx, err)
		}()
		return handler(srv, stream)
	}
}

func startTransaction(ctx context.Context, tracer *apm.Tracer, name string) (*apm.Transaction, context.Context) {
	var opts apm.TransactionOptions
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		traceContext, ok := getIncomingMetadataTraceContext(md, w3cTraceparentHeader)
		if !ok {
			traceContext, _ = getIncomingMetadataTraceContext(md, elasticTraceparentHeader)
		}
		opts.TraceContext = traceContext
	}
	tx := tracer.StartTransactionOptions(name, "request", opts)
	tx.Context.SetFramework("grpc", grpc.Version)
	if peer, ok := peer.FromContext(ctx); ok {
		// Set underlying HTTP/2.0 request context.
		//
		// https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md
		var tlsConnectionState *tls.ConnectionState
		var peerAddr string
		var authority string
		url := url.URL{Scheme: "http", Path: name}
		if info, ok := peer.AuthInfo.(credentials.TLSInfo); ok {
			url.Scheme = "https"
			tlsConnectionState = &info.State
		}
		if peer.Addr != nil {
			peerAddr = peer.Addr.String()
		}
		if values := md.Get(":authority"); len(values) > 0 {
			authority = values[0]
		}
		tx.Context.SetHTTPRequest(&http.Request{
			URL:        &url,
			Method:     "POST", // method is always POST
			ProtoMajor: 2,
			ProtoMinor: 0,
			Header:     http.Header(md),
			Host:       authority,
			RemoteAddr: peerAddr,
			TLS:        tlsConnectionState,
		})
	}
	return tx, apm.ContextWithTransaction(ctx, tx)
}

func getIncomingMetadataTraceContext(md metadata.MD, header string) (apm.TraceContext, bool) {
	if values := md.Get(header); len(values) == 1 {
		traceContext, err := apmhttp.ParseTraceparentHeader(values[0])
		if err == nil {
			traceContext.State, _ = apmhttp.ParseTracestateHeader(md.Get(tracestateHeader)...)
			return traceContext, true
		}
	}
	return apm.TraceContext{}, false
}

func setTransactionResult(tx *apm.Transaction, err error) {
	statusCode := statusCodeFromError(err)
	tx.Result = statusCode.String()

	// For gRPC servers, the transaction outcome is generally "success",
	// except for codes which are not subject to client interpretation.
	if tx.Outcome == "" {
		switch statusCode {
		case codes.Unknown,
			codes.DeadlineExceeded,
			codes.ResourceExhausted,
			codes.FailedPrecondition,
			codes.Aborted,
			codes.Internal,
			codes.Unavailable,
			codes.DataLoss:
			tx.Outcome = "failure"
		default:
			tx.Outcome = "success"
		}
	}
}

func statusCodeFromError(err error) codes.Code {
	if err == nil {
		return codes.OK
	}
	statusCode := codes.Unknown
	if s, ok := status.FromError(err); ok {
		statusCode = s.Code()
	}
	return statusCode
}

type serverOptions struct {
	tracer         *apm.Tracer
	recover        bool
	requestIgnorer RequestIgnorerFunc
	streamIgnorer  StreamIgnorerFunc
}

// ServerOption sets options for server-side tracing.
type ServerOption func(*serverOptions)

// WithTracer returns a ServerOption which sets t as the tracer
// to use for tracing server requests.
func WithTracer(t *apm.Tracer) ServerOption {
	if t == nil {
		panic("t == nil")
	}
	return func(o *serverOptions) {
		o.tracer = t
	}
}

// WithRecovery returns a ServerOption which enables panic recovery
// in the gRPC server interceptor.
//
// The interceptor will report panics as errors to Elastic APM,
// but unless this is enabled, they will still cause the server to
// be terminated. With recovery enabled, panics will be translated
// to gRPC errors with the code gprc/codes.Internal.
func WithRecovery() ServerOption {
	return func(o *serverOptions) {
		o.recover = true
	}
}

// RequestIgnorerFunc is the type of a function for use in
// WithServerRequestIgnorer.
type RequestIgnorerFunc func(*grpc.UnaryServerInfo) bool

// WithServerRequestIgnorer returns a ServerOption which sets r as the
// function to use to determine whether or not a server request should
// be ignored. If r is nil, all requests will be reported.
func WithServerRequestIgnorer(r RequestIgnorerFunc) ServerOption {
	if r == nil {
		r = IgnoreNone
	}
	return func(o *serverOptions) {
		o.requestIgnorer = r
	}
}

// StreamIgnorerFunc is the type of a function for use in
// WithServerStreamIgnorer.
type StreamIgnorerFunc func(*grpc.StreamServerInfo) bool

// WithServerStreamIgnorer returns a ServerOption which sets s as the
// function to use to determine whether or not a server stream request
// should be ignored. If s is nil, all stream requests will be reported.
func WithServerStreamIgnorer(s StreamIgnorerFunc) ServerOption {
	if s == nil {
		s = IgnoreNoneStream
	}
	return func(o *serverOptions) {
		o.streamIgnorer = s
	}
}
