package apmfasthttp

import (
	"strings"

	"github.com/savsgio/gotils/strconv"
	"github.com/valyala/fasthttp"
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

func setRequestContext(tx *Transaction, ctx *fasthttp.RequestCtx) error {
	req := &tx.req

	if err := requestCtxToRequest(ctx, req, &tx.httpBody); err != nil {
		return err
	}

	bc := tx.tracer.CaptureHTTPRequestBody(req)

	tx.body = bc
	tx.tx.Context.SetHTTPRequest(req)
	tx.tx.Context.SetHTTPRequestBody(bc)

	return nil
}

func setResponseContext(tx *Transaction) error {
	statusCode := tx.httpCtx.Response.Header.StatusCode()

	tx.tx.Result = apmhttp.StatusCodeResult(statusCode)
	if !tx.tx.Sampled() {
		return nil
	}

	headers := tx.req.Header
	resetHTTPMap(headers)

	tx.httpCtx.Response.Header.VisitAll(func(k, v []byte) {
		sk := string(k)
		sv := string(v)

		headers.Set(sk, sv)
	})

	tx.tx.Context.SetHTTPResponseHeaders(headers)
	tx.tx.Context.SetHTTPStatusCode(statusCode)

	tx.body.Discard()

	return nil
}

// StartTransaction returns a new Transaction with name,
// created with tracer, and taking trace context from ctx.
//
// If the transaction is not ignored, the request and the request body
// capturer will be returned with the transaction added to its context.
func StartTransaction(tracer *apm.Tracer, name string, ctx *fasthttp.RequestCtx) (*Transaction, error) {
	traceContext, ok := getRequestTraceparent(ctx, apmhttp.ElasticTraceparentHeader)
	if !ok {
		traceContext, ok = getRequestTraceparent(ctx, apmhttp.W3CTraceparentHeader)
	}

	if ok {
		tracestateHeader := strconv.B2S(ctx.Request.Header.Peek(apmhttp.TracestateHeader))
		traceContext.State, _ = apmhttp.ParseTracestateHeader(strings.Split(tracestateHeader, ",")...)
	}

	tx := acquireTransaction()
	tx.tracer = tracer
	tx.tx = tracer.StartTransactionOptions(name, "request", apm.TransactionOptions{TraceContext: traceContext})
	tx.httpCtx = ctx

	if err := setRequestContext(tx, ctx); err != nil {
		tx.Close()

		return nil, err
	}

	return tx, nil
}

// GetTransaction returns the transaction from the given ctx, if exist.
func GetTransaction(ctx *fasthttp.RequestCtx) *Transaction {
	if tx, ok := ctx.UserValue(TxKey).(*Transaction); ok {
		return tx
	}

	return nil
}
