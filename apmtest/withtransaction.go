package apmtest

import (
	"context"
	"fmt"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

// WithTransaction calls f with a new context containing a transaction,
// flushes the transaction to a test server, and returns the decoded
// transaction and any associated errors.
func WithTransaction(f func(ctx context.Context)) (model.Transaction, []model.Error) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tx := tracer.StartTransaction("name", "type")
	ctx := elasticapm.ContextWithTransaction(context.Background(), tx)
	f(ctx)

	tx.End()
	tracer.Flush(nil)
	payloads := transport.Payloads()
	if n := len(payloads.Transactions); n != 1 {
		panic(fmt.Errorf("expected 1 transaction, got %d", n))
	}
	return payloads.Transactions[0], payloads.Errors
}
