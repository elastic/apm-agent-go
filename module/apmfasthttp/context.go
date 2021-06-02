package apmfasthttp

import (
	"context"
	"net/http"
	"strings"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"go.elastic.co/apm"
	"go.elastic.co/apm/internal/apmcontext"
	"go.elastic.co/apm/module/apmhttp"
)

func init() {
	origTransactionFromContext := apmcontext.TransactionFromContext
	apmcontext.TransactionFromContext = func(ctx context.Context) interface{} {
		if tx, ok := ctx.Value(txKey).(*TxCloser); ok {
			return tx.Tx()
		}

		return origTransactionFromContext(ctx)
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

func setResponseContext(ctx *fasthttp.RequestCtx, tx *apm.Transaction, bc *apm.BodyCapturer) error {
	statusCode := ctx.Response.Header.StatusCode()

	tx.Result = apmhttp.StatusCodeResult(statusCode)
	if !tx.Sampled() {
		return nil
	}

	headers := make(http.Header)
	ctx.Response.Header.VisitAll(func(k, v []byte) {
		sk := string(k)
		sv := string(v)

		headers.Set(sk, sv)
	})

	tx.Context.SetHTTPResponseHeaders(headers)
	tx.Context.SetHTTPStatusCode(statusCode)

	if bc != nil {
		bc.Discard()
	}

	return nil
}

// StartTransactionWithBody returns a new Transaction with name,
// created with tracer, and taking trace context from ctx.
//
// If the transaction is not ignored, the request and the request body
// capturer will be returned with the transaction added to its context.
func StartTransactionWithBody(
	tracer *apm.Tracer, name string, ctx *fasthttp.RequestCtx,
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

	return tx, bc, nil
}
