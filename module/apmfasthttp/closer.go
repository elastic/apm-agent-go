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

//go:build go1.12
// +build go1.12

package apmfasthttp // import "go.elastic.co/apm/module/apmfasthttp"

import (
	"go.elastic.co/apm"
)

// newTxCloser returns a transaction closer.
func newTxCloser(tx *apm.Transaction, bc *apm.BodyCapturer) *txCloser {
	return &txCloser{
		tx: tx,
		bc: bc,
	}
}

// Close ends the transaction. The closer exists because, while the wrapped
// handler may return, an underlying streaming body writer may still exist.
// References to the transaction and bodycloser are held in a txCloser struct,
// which is added to the context on the request. From the fasthttp
// documentation:
// All the values are removed from ctx after returning from the top
// RequestHandler. Additionally, Close method is called on each value
// implementing io.Closer before removing the value from ctx.
func (c *txCloser) Close() error {
	c.tx.End()
	c.bc.Discard()
	return nil
}
