// +build go1.9

package apmgrpc

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"go.elastic.co/apm"
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
		span, ctx := apm.StartSpan(ctx, method, "grpc")
		defer span.End()
		return invoker(ctx, method, req, resp, cc, opts...)
	}
}

type clientOptions struct {
	tracer *apm.Tracer
}

// ClientOption sets options for client-side tracing.
type ClientOption func(*clientOptions)
