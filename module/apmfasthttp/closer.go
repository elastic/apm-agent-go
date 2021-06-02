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

package apmfasthttp

import (
	"github.com/valyala/fasthttp"

	"go.elastic.co/apm"
)

// NewTxCloser returns a transaction closer.
func NewTxCloser(ctx *fasthttp.RequestCtx, tx *apm.Transaction, bc *apm.BodyCapturer) *TxCloser {
	return &TxCloser{
		ctx: ctx,
		tx:  tx,
		bc:  bc,
	}
}

// Tx returns the underlying APM transaction.
func (c *TxCloser) Tx() *apm.Transaction {
	return c.tx
}

// Tx returns the underlying APM body capturer.
func (c *TxCloser) BodyCapturer() *apm.BodyCapturer {
	return c.bc
}

// Close sets the response context to the APM transaction and
// ends the transaction.
func (c *TxCloser) Close() error {
	if err := setResponseContext(c.ctx, c.tx, c.bc); err != nil {
		return err
	}

	c.tx.End()

	return nil
}
