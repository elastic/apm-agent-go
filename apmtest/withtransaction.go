package apmtest

import (
	"context"
	"fmt"

	"go.elastic.co/apm"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/transport/transporttest"
)

// WithTransaction calls f with a new context containing a transaction,
// flushes the transaction to a test server, and returns the decoded
// transaction and any associated errors.
func WithTransaction(f func(ctx context.Context)) (model.Transaction, []model.Span, []model.Error) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tx := tracer.StartTransaction("name", "type")
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	f(ctx)

	tx.End()
	tracer.Flush(nil)
	payloads := transport.Payloads()
	if n := len(payloads.Transactions); n != 1 {
		panic(fmt.Errorf("expected 1 transaction, got %d", n))
	}
	return payloads.Transactions[0], payloads.Spans, payloads.Errors
}
