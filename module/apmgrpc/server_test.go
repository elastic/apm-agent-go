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

// +build go1.9

package apmgrpc_test

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"
	"testing"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"go.elastic.co/apm"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmgrpc"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/stacktrace"
	"go.elastic.co/apm/transport/transporttest"
)

func init() {
	// Register this test package as an application package so we can
	// check "culprit".
	type foo struct{}
	stacktrace.RegisterApplicationPackage(reflect.TypeOf(foo{}).PkgPath())
}

func TestServerTransaction(t *testing.T) {
	adaptTest := func(f func(*testing.T, testParams)) func(*testing.T) {
		return func(t *testing.T) {
			tracer, transport := transporttest.NewRecorderTracer()
			defer tracer.Close()

			s, server, addr := newServer(t, tracer)
			defer s.GracefulStop()

			conn, client := newClient(t, addr)
			defer conn.Close()

			f(t, testParams{
				server:    server,
				conn:      conn,
				client:    client,
				tracer:    tracer,
				transport: transport,
			})
		}
	}
	t.Run("happy", adaptTest(testServerTransactionHappy))
	t.Run("unknown_error", adaptTest(testServerTransactionUnknownError))
	t.Run("status_error", adaptTest(testServerTransactionStatusError))
	t.Run("panic", adaptTest(testServerTransactionPanic))
}

type testParams struct {
	server    *helloworldServer
	conn      *grpc.ClientConn
	client    pb.GreeterClient
	tracer    *apm.Tracer
	transport *transporttest.RecorderTransport
}

func testServerTransactionHappy(t *testing.T, p testParams) {
	traceID := apm.TraceID{0x0a, 0xf7, 0x65, 0x19, 0x16, 0xcd, 0x43, 0xdd, 0x84, 0x48, 0xeb, 0x21, 0x1c, 0x80, 0x31, 0x9c}
	clientSpanID := apm.SpanID{0xb7, 0xad, 0x6b, 0x71, 0x69, 0x20, 0x33, 0x31}

	traceparentValue := fmt.Sprintf("00-%s-%s-01", traceID, clientSpanID)

	headers := []string{apmhttp.ElasticTraceparentHeader, apmhttp.W3CTraceparentHeader}
	for _, header := range headers {
		ctx := metadata.AppendToOutgoingContext(context.Background(), header, traceparentValue)
		resp, err := p.client.SayHello(ctx, &pb.HelloRequest{Name: "birita"})
		require.NoError(t, err)
		assert.Equal(t, resp, &pb.HelloReply{Message: "hello, birita"})
	}
	p.tracer.Flush(nil)
	payloads := p.transport.Payloads()
	require.Len(t, payloads.Transactions, len(headers))

	for i, tx := range payloads.Transactions {
		assert.Equal(t, "/helloworld.Greeter/SayHello", tx.Name)
		assert.Equal(t, "request", tx.Type)
		assert.Equal(t, "OK", tx.Result)
		assert.Equal(t, model.TraceID(traceID), tx.TraceID)
		assert.Equal(t, model.SpanID(clientSpanID), tx.ParentID)
		assert.Equal(t, &model.Context{
			Service: &model.Service{
				Framework: &model.Framework{
					Name:    "grpc",
					Version: grpc.Version,
				},
			},
			Custom: model.IfaceMap{{
				Key:   strings.ToLower(headers[i]),
				Value: traceparentValue,
			}},
		}, tx.Context)
	}
}

func testServerTransactionUnknownError(t *testing.T, p testParams) {
	p.server.err = errors.New("boom")
	_, err := p.client.SayHello(context.Background(), &pb.HelloRequest{Name: "birita"})
	assert.EqualError(t, err, "rpc error: code = Unknown desc = boom")

	p.tracer.Flush(nil)
	payloads := p.transport.Payloads()
	tx := payloads.Transactions[0]
	assert.Equal(t, "/helloworld.Greeter/SayHello", tx.Name)
	assert.Equal(t, "request", tx.Type)
	assert.Equal(t, "Unknown", tx.Result)
}

func testServerTransactionStatusError(t *testing.T, p testParams) {
	p.server.err = status.Errorf(codes.DataLoss, "boom")
	_, err := p.client.SayHello(context.Background(), &pb.HelloRequest{Name: "birita"})
	assert.EqualError(t, err, "rpc error: code = DataLoss desc = boom")

	p.tracer.Flush(nil)
	payloads := p.transport.Payloads()
	tx := payloads.Transactions[0]
	assert.Equal(t, "/helloworld.Greeter/SayHello", tx.Name)
	assert.Equal(t, "request", tx.Type)
	assert.Equal(t, "DataLoss", tx.Result)
}

func testServerTransactionPanic(t *testing.T, p testParams) {
	p.server.panic = true
	p.server.err = errors.New("boom")
	_, err := p.client.SayHello(context.Background(), &pb.HelloRequest{Name: "birita"})
	assert.EqualError(t, err, "rpc error: code = Internal desc = boom")

	p.tracer.Flush(nil)
	payloads := p.transport.Payloads()
	e := payloads.Errors[0]
	assert.NotEmpty(t, e.TransactionID)
	assert.Equal(t, false, e.Exception.Handled)
	assert.Equal(t, "(*helloworldServer).SayHello", e.Culprit)
	assert.Equal(t, "boom", e.Exception.Message)
}

func TestServerRecovery(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	s, server, addr := newServer(t, tracer, apmgrpc.WithRecovery())
	defer s.GracefulStop()

	conn, client := newClient(t, addr)
	defer conn.Close()

	server.panic = true
	server.err = errors.New("boom")
	_, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: "birita"})
	assert.EqualError(t, err, "rpc error: code = Internal desc = boom")

	tracer.Flush(nil)
	payloads := transport.Payloads()
	e := payloads.Errors[0]
	assert.NotEmpty(t, e.TransactionID)

	// Panic was recovered by the recovery interceptor and translated
	// into an Internal error.
	assert.Equal(t, true, e.Exception.Handled)
	assert.Equal(t, "(*helloworldServer).SayHello", e.Culprit)
	assert.Equal(t, "boom", e.Exception.Message)
}

func TestServerIgnorer(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	s, _, addr := newServer(t, tracer, apmgrpc.WithRecovery(), apmgrpc.WithServerRequestIgnorer(func(*grpc.UnaryServerInfo) bool {
		return true
	}))
	defer s.GracefulStop()

	conn, client := newClient(t, addr)
	defer conn.Close()

	resp, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: "birita"})
	require.NoError(t, err)
	assert.Equal(t, resp, &pb.HelloReply{Message: "hello, birita"})

	tracer.Flush(nil)
	assert.Empty(t, transport.Payloads())
}

func newServer(t *testing.T, tracer *apm.Tracer, opts ...apmgrpc.ServerOption) (*grpc.Server, *helloworldServer, net.Addr) {
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
	server := &helloworldServer{}
	pb.RegisterGreeterServer(s, server)
	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	go s.Serve(lis)
	return s, server, lis.Addr()
}

func newClient(t *testing.T, addr net.Addr) (*grpc.ClientConn, pb.GreeterClient) {
	conn, err := grpc.Dial(
		addr.String(), grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(apmgrpc.NewUnaryClientInterceptor()),
	)
	require.NoError(t, err)
	return conn, pb.NewGreeterClient(conn)
}

type helloworldServer struct {
	panic bool
	err   error
}

func (s *helloworldServer) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	// The context passed to the server should contain a Transaction for the gRPC request.
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		tx := apm.TransactionFromContext(ctx)
		for _, header := range []string{"elastic-apm-traceparent", "traceparent", "tracestate"} {
			if values := md.Get(header); len(values) > 0 {
				tx.Context.SetCustom(header, strings.Join(values, " "))
			}
		}
	}
	span, ctx := apm.StartSpan(ctx, "server_span", "type")
	if tracestate := span.TraceContext().State.String(); tracestate != "" {
		span.Name = tracestate
	}
	span.End()
	if s.panic {
		panic(s.err)
	}
	if s.err != nil {
		return nil, s.err
	}
	return &pb.HelloReply{Message: "hello, " + req.Name}, nil
}
