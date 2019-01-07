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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/transport/transporttest"
)

func TestClientSpan(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	serverTracer, serverTransport := transporttest.NewRecorderTracer()
	defer serverTracer.Close()
	s, _, addr := newServer(t, serverTracer)
	defer s.GracefulStop()

	conn, client := newClient(t, addr)
	defer conn.Close()
	resp, err := client.SayHello(context.Background(), &pb.HelloRequest{Name: "birita"})
	require.NoError(t, err)
	assert.Equal(t, resp, &pb.HelloReply{Message: "hello, birita"})

	// The client interceptor starts no transactions, only spans.
	tracer.Flush(nil)
	require.Zero(t, transport.Payloads())

	_, clientSpans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		resp, err = client.SayHello(ctx, &pb.HelloRequest{Name: "birita"})
		require.NoError(t, err)
		assert.Equal(t, resp, &pb.HelloReply{Message: "hello, birita"})
	})

	require.Len(t, clientSpans, 1)
	assert.Equal(t, "/helloworld.Greeter/SayHello", clientSpans[0].Name)
	assert.Equal(t, "external", clientSpans[0].Type)
	assert.Equal(t, "grpc", clientSpans[0].Subtype)

	serverTracer.Flush(nil)
	serverTransactions := serverTransport.Payloads().Transactions
	require.Len(t, serverTransactions, 2)
	for _, tx := range serverTransactions {
		assert.Equal(t, "/helloworld.Greeter/SayHello", tx.Name)
	}
	assert.Equal(t, clientSpans[0].TraceID, serverTransactions[1].TraceID)
	assert.Equal(t, clientSpans[0].ID, serverTransactions[1].ParentID)
}
