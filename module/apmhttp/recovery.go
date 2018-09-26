package apmhttp

import (
	"net/http"

	"github.com/elastic/apm-agent-go"
)

// RecoveryFunc is the type of a function for use in WithRecovery.
type RecoveryFunc func(
	w http.ResponseWriter,
	req *http.Request,
	resp *Response,
	body *elasticapm.BodyCapturer,
	tx *elasticapm.Transaction,
	recovered interface{},
)

// NewTraceRecovery returns a RecoveryFunc for use in WithRecovery.
//
// The returned RecoveryFunc will report recovered error to Elastic APM
// using the given Tracer, or elasticapm.DefaultTracer if t is nil. The
// error will be linked to the given transaction.
func NewTraceRecovery(t *elasticapm.Tracer) RecoveryFunc {
	if t == nil {
		t = elasticapm.DefaultTracer
	}
	return func(
		w http.ResponseWriter,
		req *http.Request,
		resp *Response,
		body *elasticapm.BodyCapturer,
		tx *elasticapm.Transaction,
		recovered interface{},
	) {
		e := t.Recovered(recovered)
		e.SetTransaction(tx)
		setContext(&e.Context, req, resp, body)
		e.Send()
	}
}
