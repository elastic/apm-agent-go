package apmfasthttp

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

	return func(ctx *fasthttp.RequestCtx, tx *Transaction, recovered interface{}) {
		e := t.Recovered(recovered)
		e.SetTransaction(tx.Tx())

		_ = setResponseContext(tx)

		e.Send()
	}
}
