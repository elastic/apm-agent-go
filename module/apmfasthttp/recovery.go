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
	"github.com/valyala/fasthttp"

	"go.elastic.co/apm"
)

// NewTraceRecovery returns a RecoveryFunc for use in WithRecovery.
//
// The returned RecoveryFunc will report recovered error to Elastic APM
// using the given Tracer, or apm.DefaultTracer if t is nil. The
// error will be linked to the given transaction.
//
// If headers have not already been written, a 500 response will be sent.
func NewTraceRecovery(t *apm.Tracer) RecoveryFunc {
	if t == nil {
		t = apm.DefaultTracer
	}

	return func(ctx *fasthttp.RequestCtx, tx *apm.Transaction, bc *apm.BodyCapturer, recovered interface{}) {
		_ = setResponseContext(ctx, tx, bc)

		e := t.Recovered(recovered)
		e.SetTransaction(tx)
		e.Send()
	}
}
