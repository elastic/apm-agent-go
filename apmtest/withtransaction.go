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

package apmtest

import (
	"context"
	"fmt"

	"go.elastic.co/apm"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/transport/transporttest"
)

// WithTransaction is equivalent to calling WithTransactionOptions with a zero TransactionOptions.
func WithTransaction(f func(ctx context.Context)) (model.Transaction, []model.Span, []model.Error) {
	return WithTransactionOptions(apm.TransactionOptions{}, f)
}

// WithTransactionOptions calls f with a new context containing a transaction
// and transaction options, flushes the transaction to a test server, and returns
// the decoded transaction and any associated spans and errors.
func WithTransactionOptions(opts apm.TransactionOptions, f func(ctx context.Context)) (model.Transaction, []model.Span, []model.Error) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tx := tracer.StartTransactionOptions("name", "type", opts)
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
