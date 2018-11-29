// +build go1.9

package apmgrpc_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"

	"go.elastic.co/apm"
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

	tx := tracer.StartTransaction("name", "type")
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	resp, err = client.SayHello(ctx, &pb.HelloRequest{Name: "birita"})
	require.NoError(t, err)
	assert.Equal(t, resp, &pb.HelloReply{Message: "hello, birita"})
	tx.End()

	tracer.Flush(nil)
	clientSpans := transport.Payloads().Spans
	require.Len(t, clientSpans, 1)
	assert.Equal(t, "/helloworld.Greeter/SayHello", clientSpans[0].Name)
	assert.Equal(t, "external.grpc", clientSpans[0].Type)

	serverTracer.Flush(nil)
	serverTransactions := serverTransport.Payloads().Transactions
	require.Len(t, serverTransactions, 2)
	for _, tx := range serverTransactions {
		assert.Equal(t, "/helloworld.Greeter/SayHello", tx.Name)
	}
	assert.Equal(t, clientSpans[0].TraceID, serverTransactions[1].TraceID)
	assert.Equal(t, clientSpans[0].ID, serverTransactions[1].ParentID)
}
