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

package apmgrpc_test

import (
	"io"
	"net"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/status"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmgrpc/internal/testservice"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/transport/transporttest"
)

func TestClientSpan(t *testing.T) {
	t.Run("with-elastic-apm-traceparent", func(t *testing.T) {
		testClientSpan(t, "elastic-apm-traceparent", "traceparent")
	})
	t.Run("without-elastic-apm-traceparent", func(t *testing.T) {
		os.Setenv("ELASTIC_APM_USE_ELASTIC_TRACEPARENT_HEADER", "false")
		defer os.Unsetenv("ELASTIC_APM_USE_ELASTIC_TRACEPARENT_HEADER")
		testClientSpan(t, "traceparent")
	})
}

func testClientSpan(t *testing.T, traceparentHeaders ...string) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	serverTracer, serverTransport := transporttest.NewRecorderTracer()
	defer serverTracer.Close()
	s, _, addr := newGreeterServer(t, serverTracer)
	defer s.GracefulStop()
	tcpAddr := addr.(*net.TCPAddr)

	conn, client := newGreeterClient(t, addr)
	defer conn.Close()
	resp, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: "birita"})
	require.NoError(t, err)
	assert.Equal(t, resp, &pb.HelloReply{Message: "hello, birita"})

	// The client interceptor starts no transactions, only spans.
	tracer.Flush(nil)
	require.Zero(t, transport.Payloads())

	_, clientSpans, _ := apmtest.WithTransactionOptions(apm.TransactionOptions{
		TraceContext: apm.TraceContext{
			Trace:   apm.TraceID{1},
			Span:    apm.SpanID{1},
			Options: apm.TraceOptions(0).WithRecorded(true),
			State:   apm.NewTraceState(apm.TraceStateEntry{Key: "vendor", Value: "tracestate"}),
		},
	}, func(ctx context.Context) {
		resp, err = client.SayHello(ctx, &pb.HelloRequest{Name: "birita"})
		require.NoError(t, err)
		assert.Equal(t, resp, &pb.HelloReply{Message: "hello, birita"})
	})

	require.Len(t, clientSpans, 1)
	assert.Equal(t, "/helloworld.Greeter/SayHello", clientSpans[0].Name)
	assert.Equal(t, "external", clientSpans[0].Type)
	assert.Equal(t, "grpc", clientSpans[0].Subtype)
	assert.Equal(t, &model.SpanContext{
		HTTP: &model.HTTPSpanContext{
			URL: &url.URL{
				Scheme: "http",
				Host:   tcpAddr.String(),
				Path:   "/helloworld.Greeter/SayHello",
			},
		},
		Destination: &model.DestinationSpanContext{
			Address: tcpAddr.IP.String(),
			Port:    tcpAddr.Port,
			Service: &model.DestinationServiceSpanContext{
				Type:     "external",
				Name:     tcpAddr.String(),
				Resource: tcpAddr.String(),
			},
		},
	}, clientSpans[0].Context)

	serverTracer.Flush(nil)
	serverTransactions := serverTransport.Payloads().Transactions
	serverSpans := serverTransport.Payloads().Spans
	require.Len(t, serverTransactions, 2)
	require.Len(t, serverSpans, 2)
	for _, tx := range serverTransactions {
		assert.Equal(t, "/helloworld.Greeter/SayHello", tx.Name)
	}
	assert.Equal(t, clientSpans[0].TraceID, serverTransactions[1].TraceID)
	assert.Equal(t, clientSpans[0].ID, serverTransactions[1].ParentID)
	assert.Equal(t, "es=s:1", serverSpans[0].Name) // automatically created tracestate
	assert.Equal(t, "vendor=tracestate", serverSpans[1].Name)

	traceparentValue := apmhttp.FormatTraceparentHeader(apm.TraceContext{
		Trace:   apm.TraceID(clientSpans[0].TraceID),
		Span:    apm.SpanID(clientSpans[0].ID),
		Options: apm.TraceOptions(0).WithRecorded(true),
	})
	expectedCustom := model.IfaceMap{}
	for _, header := range traceparentHeaders {
		expectedCustom = append(expectedCustom, model.IfaceMapItem{
			Key:   header,
			Value: traceparentValue,
		})
	}
	expectedCustom = append(expectedCustom, model.IfaceMapItem{
		Key:   "tracestate",
		Value: "vendor=tracestate",
	})
	assert.Equal(t, expectedCustom, serverTransactions[1].Context.Custom)
}

func TestClientSpanDropped(t *testing.T) {
	serverTracer := apmtest.NewRecordingTracer()
	defer serverTracer.Close()
	s, _, addr := newGreeterServer(t, serverTracer.Tracer)
	defer s.GracefulStop()

	conn, client := newGreeterClient(t, addr)
	defer conn.Close()

	clientTracer := apmtest.NewRecordingTracer()
	defer clientTracer.Close()
	clientTracer.SetMaxSpans(1)

	clientTransaction, clientSpans, _ := clientTracer.WithTransaction(func(ctx context.Context) {
		for i := 0; i < 2; i++ {
			_, err := client.SayHello(ctx, &pb.HelloRequest{Name: "birita"})
			require.NoError(t, err)
		}
	})
	require.Len(t, clientSpans, 1)

	serverTracer.Flush(nil)
	serverTransactions := serverTracer.Payloads().Transactions
	require.Len(t, serverTransactions, 2)
	for _, serverTransaction := range serverTransactions {
		assert.Equal(t, clientTransaction.TraceID, serverTransaction.TraceID)
	}
	assert.Equal(t, clientSpans[0].ID, serverTransactions[0].ParentID)
	assert.Equal(t, clientTransaction.ID, serverTransactions[1].ParentID)
}

func TestClientTransactionUnsampled(t *testing.T) {
	serverTracer := apmtest.NewRecordingTracer()
	defer serverTracer.Close()
	s, _, addr := newGreeterServer(t, serverTracer.Tracer)
	defer s.GracefulStop()

	conn, client := newGreeterClient(t, addr)
	defer conn.Close()

	clientTracer := apmtest.NewRecordingTracer()
	defer clientTracer.Close()
	clientTracer.SetSampler(apm.NewRatioSampler(0)) // sample nothing

	clientTransaction, clientSpans, _ := clientTracer.WithTransaction(func(ctx context.Context) {
		_, err := client.SayHello(ctx, &pb.HelloRequest{Name: "birita"})
		require.NoError(t, err)
	})
	require.Len(t, clientSpans, 0)

	serverTracer.Flush(nil)
	serverTransactions := serverTracer.Payloads().Transactions
	require.Len(t, serverTransactions, 1)
	assert.Equal(t, clientTransaction.TraceID, serverTransactions[0].TraceID)
	assert.Equal(t, clientTransaction.ID, serverTransactions[0].ParentID)
}

func TestClientOutcome(t *testing.T) {
	s, helloworldServer, addr := newGreeterServer(t, apmtest.DiscardTracer)
	defer s.GracefulStop()

	conn, client := newGreeterClient(t, addr)
	defer conn.Close()

	clientTracer := apmtest.NewRecordingTracer()
	defer clientTracer.Close()

	_, spans, _ := clientTracer.WithTransaction(func(ctx context.Context) {
		client.SayHello(ctx, &pb.HelloRequest{Name: "birita"})

		helloworldServer.err = status.Errorf(codes.Unknown, "boom")
		client.SayHello(ctx, &pb.HelloRequest{Name: "birita"})

		helloworldServer.err = status.Errorf(codes.NotFound, "boom")
		client.SayHello(ctx, &pb.HelloRequest{Name: "birita"})
	})
	require.Len(t, spans, 3)

	assert.Equal(t, "success", spans[0].Outcome)
	assert.Equal(t, "failure", spans[1].Outcome) // unknown error
	assert.Equal(t, "failure", spans[2].Outcome)
}

func TestStreamClientSpan(t *testing.T) {
	clientTracer, clientTransport := transporttest.NewRecorderTracer()
	defer clientTracer.Close()

	serverTracer, serverTransport := transporttest.NewRecorderTracer()
	defer serverTracer.Close()
	s, _, addr := newAccumulatorServer(t, serverTracer)
	defer s.GracefulStop()

	conn, client := newAccumulatorClient(t, addr)
	defer conn.Close()

	clientTransaction := clientTracer.StartTransaction("name", "type")
	ctx := apm.ContextWithTransaction(context.Background(), clientTransaction)

	stream, err := client.Accumulate(ctx)
	require.NoError(t, err)
	err = stream.Send(&testservice.AccumulateRequest{Value: 123})
	require.NoError(t, err)
	reply, err := stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, int64(123), reply.Value)

	err = stream.CloseSend()
	require.NoError(t, err)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)

	timeout := time.NewTimer(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer timeout.Stop()
	defer ticker.Stop()
	var done bool
	for !done {
		select {
		case <-ticker.C:
			clientTracer.Flush(nil)
			if len(clientTransport.Payloads().Spans) > 0 {
				done = true
			}
		case <-timeout.C:
			t.Fatal("timed out waiting for client span to end")
		}
	}
	clientTransaction.End()

	clientTracer.Flush(nil)
	clientPayloads := clientTransport.Payloads()
	require.Len(t, clientPayloads.Transactions, 1)
	require.Len(t, clientPayloads.Spans, 1)
	assert.Equal(t, "/go.elastic.co.apm.module.apmgrpc.testservice.Accumulator/Accumulate", clientPayloads.Spans[0].Name)
	assert.Equal(t, "external", clientPayloads.Spans[0].Type)
	assert.Equal(t, "grpc", clientPayloads.Spans[0].Subtype)

	serverTracer.Flush(nil)
	serverPayloads := serverTransport.Payloads()
	require.Len(t, serverPayloads.Transactions, 1)
	assert.Equal(t, clientPayloads.Spans[0].ID, serverPayloads.Transactions[0].ParentID)
	assert.Equal(t, clientPayloads.Spans[0].TraceID, serverPayloads.Transactions[0].TraceID)
}
