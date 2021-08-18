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

package apmgokit_test

import (
	"context"
	"net"
	"testing"

	kitgrpc "github.com/go-kit/kit/transport/grpc"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	netcontext "golang.org/x/net/context"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/module/apmgrpc"
	"go.elastic.co/apm/transport/transporttest"
)

func Example_grpcServer() {
	// Create your go-kit/kit/transport/grpc.Server as usual, without any tracing middleware.
	endpoint := func(ctx context.Context, req interface{}) (interface{}, error) {
		// The middleware added to the underlying gRPC server will be propagate
		// a transaction to the context passed to your endpoint. You can then
		// report endpoint-specific spans using apm.StartSpan.
		span, ctx := apm.StartSpan(ctx, "name", "endpoint")
		defer span.End()
		return nil, nil
	}
	var encodeRequest func(ctx context.Context, req interface{}) (interface{}, error)
	var decodeResponse func(ctx context.Context, req interface{}) (interface{}, error)
	service := &helloWorldService{kitgrpc.NewServer(
		endpoint,
		encodeRequest,
		decodeResponse,
	)}

	// When creating the underlying gRPC server, use the apmgrpc.NewUnaryServerInterceptor
	// function (from module/apmgrpc). This will trace all incoming requests.
	s := grpc.NewServer(grpc.UnaryInterceptor(apmgrpc.NewUnaryServerInterceptor()))
	defer s.GracefulStop()
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}
	go s.Serve(lis)
	pb.RegisterGreeterServer(s, service)
}

func Example_grpcClient() {
	// When dialling the gRPC client connection, use the apmgrpc.NewUnaryClientInterceptor
	// function (from module/apmgrpc). This will trace all outgoing requests, as long as
	// the context supplied to methods include an apm.Transaction.
	conn, err := grpc.Dial("localhost:1234", grpc.WithUnaryInterceptor(apmgrpc.NewUnaryClientInterceptor()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Create your go-kit/kit/transport/grpc.Client as usual, without any tracing middleware.
	client := kitgrpc.NewClient(
		conn, "helloworld.Greeter", "SayHello",
		func(ctx context.Context, req interface{}) (interface{}, error) {
			return &pb.HelloRequest{Name: req.(string)}, nil
		},
		func(ctx context.Context, resp interface{}) (interface{}, error) {
			return resp, nil
		},
		&pb.HelloReply{},
	)

	tx := apm.DefaultTracer.StartTransaction("name", "type")
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	defer tx.End()

	_, err = client.Endpoint()(ctx, "world")
	if err != nil {
		panic(err)
	}
}

func TestGRPCTransport(t *testing.T) {
	serverTracer, serverTransport := transporttest.NewRecorderTracer()
	defer serverTracer.Close()

	sayHelloEndpoint := func(ctx context.Context, req interface{}) (interface{}, error) {
		span, ctx := apm.StartSpan(ctx, "SayHello", "endpoint")
		defer span.End()
		return nil, nil
	}

	s, addr := newServer(t, serverTracer, &helloWorldService{
		sayHello: kitgrpc.NewServer(
			sayHelloEndpoint,
			func(ctx context.Context, req interface{}) (interface{}, error) {
				return req, nil
			},
			func(ctx context.Context, resp interface{}) (interface{}, error) {
				return &pb.HelloReply{}, nil
			},
		),
	})
	defer s.GracefulStop()

	conn := newClient(t, addr)
	defer conn.Close()

	client := kitgrpc.NewClient(
		conn, "helloworld.Greeter", "SayHello",
		func(ctx context.Context, req interface{}) (interface{}, error) {
			return &pb.HelloRequest{Name: req.(string)}, nil
		},
		func(ctx context.Context, resp interface{}) (interface{}, error) {
			return &pb.HelloReply{}, nil
		},
		&pb.HelloReply{},
	)
	_, clientSpans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		_, err := client.Endpoint()(ctx, "birita")
		assert.NoError(t, err)
	})
	require.Len(t, clientSpans, 1)
	assert.Equal(t, "/helloworld.Greeter/SayHello", clientSpans[0].Name)

	serverTracer.Flush(nil)
	payloads := serverTransport.Payloads()
	require.Len(t, payloads.Transactions, 1)
	require.Len(t, payloads.Spans, 1)
	assert.Equal(t, "/helloworld.Greeter/SayHello", payloads.Transactions[0].Name)
	assert.Equal(t, "endpoint", payloads.Spans[0].Type)
	assert.Equal(t, clientSpans[0].ID, payloads.Transactions[0].ParentID)
	assert.Equal(t, clientSpans[0].TraceID, payloads.Transactions[0].TraceID)
}

type helloWorldService struct {
	sayHello *kitgrpc.Server
}

func (s *helloWorldService) SayHello(ctx netcontext.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	_, rep, err := s.sayHello.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.HelloReply), nil
}

func newServer(t *testing.T, tracer *apm.Tracer, server pb.GreeterServer, opts ...apmgrpc.ServerOption) (*grpc.Server, net.Addr) {
	// We always install grpc_recovery first to avoid panics
	// aborting the test process. We install it before the
	// apmgrpc interceptor so that apmgrpc can recover panics
	// itself if configured to do so.
	interceptors := []grpc.UnaryServerInterceptor{grpc_recovery.UnaryServerInterceptor()}
	serverOpts := []grpc.ServerOption{}
	if tracer != nil {
		opts = append(opts, apmgrpc.WithTracer(tracer))
		interceptors = append(interceptors, apmgrpc.NewUnaryServerInterceptor(opts...))
	}
	serverOpts = append(serverOpts, grpc_middleware.WithUnaryServerChain(interceptors...))

	s := grpc.NewServer(serverOpts...)
	pb.RegisterGreeterServer(s, server)
	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	go s.Serve(lis)
	return s, lis.Addr()
}

func newClient(t *testing.T, addr net.Addr) *grpc.ClientConn {
	conn, err := grpc.Dial(
		addr.String(), grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(apmgrpc.NewUnaryClientInterceptor()),
	)
	require.NoError(t, err)
	return conn
}
