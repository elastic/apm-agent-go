// +build go1.9

package apmgrpc

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

// NewUnaryClientInterceptor returns a grpc.UnaryClientInterceptor that
// traces gRPC requests with the given options.
//
// The interceptor will trace spans with the "grpc" type for each request
// made, for any client method presented with a context containing a sampled
// apm.Transaction.
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
		span, ctx := startSpan(ctx, method)
		defer span.End()
		return invoker(ctx, method, req, resp, cc, opts...)
	}
}

func startSpan(ctx context.Context, name string) (*apm.Span, context.Context) {
	span, ctx := apm.StartSpan(ctx, name, "external.grpc")
	if span.Dropped() {
		return span, ctx
	}
	traceparentValue := apmhttp.FormatTraceparentHeader(span.TraceContext())
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.Pairs(traceparentHeader, traceparentValue)
	} else {
		md = md.Copy()
		md.Set(traceparentHeader, traceparentValue)
	}
	return span, metadata.NewOutgoingContext(ctx, md)
}

type clientOptions struct {
	tracer *apm.Tracer
}

// ClientOption sets options for client-side tracing.
type ClientOption func(*clientOptions)
