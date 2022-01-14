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

package apmfasthttp // import "go.elastic.co/apm/module/apmfasthttp"

import (
	"context"
	"net/http"
	"strings"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

const txKey = "apmfasthttp_transaction"

func init() {
	origTransactionFromContext := apm.OverrideTransactionFromContext
	apm.OverrideTransactionFromContext = func(ctx context.Context) *apm.Transaction {
		if tx, ok := ctx.Value(txKey).(*txCloser); ok {
			return tx.tx
		}
		return origTransactionFromContext(ctx)
	}

	origBodyCapturerFromContext := apm.OverrideBodyCapturerFromContext
	apm.OverrideBodyCapturerFromContext = func(ctx context.Context) *apm.BodyCapturer {
		if tx, ok := ctx.Value(txKey).(*txCloser); ok {
			return tx.bc
		}
		return origBodyCapturerFromContext(ctx)
	}
}

func setRequestContext(ctx *fasthttp.RequestCtx, tracer *apm.Tracer, tx *apm.Transaction) (*apm.BodyCapturer, error) {
	req := new(http.Request)
	if err := fasthttpadaptor.ConvertRequest(ctx, req, true); err != nil {
		return nil, err
	}

	bc := tracer.CaptureHTTPRequestBody(req)
	tx.Context.SetHTTPRequest(req)
	tx.Context.SetHTTPRequestBody(bc)

	return bc, nil
}

func setResponseContext(ctx *fasthttp.RequestCtx, tx *apm.Transaction, bc *apm.BodyCapturer) {
	statusCode := ctx.Response.Header.StatusCode()

	tx.Result = apmhttp.StatusCodeResult(statusCode)
	if !tx.Sampled() {
		return
	}

	headers := make(http.Header)
	ctx.Response.Header.VisitAll(func(k, v []byte) {
		sk := string(k)
		sv := string(v)

		headers.Set(sk, sv)
	})

	tx.Context.SetHTTPResponseHeaders(headers)
	tx.Context.SetHTTPStatusCode(statusCode)

	return
}

// StartTransactionWithBody returns a new Transaction with name,
// created with tracer, and taking trace context from ctx.
//
// If the transaction is not ignored, the request and the request body
// capturer will be returned with the transaction added to its context.
func StartTransactionWithBody(
	ctx *fasthttp.RequestCtx, tracer *apm.Tracer, name string,
) (*apm.Transaction, *apm.BodyCapturer, error) {
	traceContext, ok := getRequestTraceparent(ctx, apmhttp.W3CTraceparentHeader)
	if !ok {
		traceContext, ok = getRequestTraceparent(ctx, apmhttp.ElasticTraceparentHeader)
	}

	if ok {
		tracestateHeader := string(ctx.Request.Header.Peek(apmhttp.TracestateHeader))
		traceContext.State, _ = apmhttp.ParseTracestateHeader(strings.Split(tracestateHeader, ",")...)
	}

	tx := tracer.StartTransactionOptions(name, "request", apm.TransactionOptions{TraceContext: traceContext})

	bc, err := setRequestContext(ctx, tracer, tx)
	if err != nil {
		tx.End()

		return nil, nil, err
	}

	ctx.SetUserValue(txKey, newTxCloser(tx, bc))

	return tx, bc, nil
}
