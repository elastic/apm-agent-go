package apmfasthttp

import (
	"net/http"

	"github.com/valyala/fasthttp"
	"go.elastic.co/apm"
)

type apmHandler struct {
	requestHandler   fasthttp.RequestHandler
	tracer           *apm.Tracer
	requestName      RequestNameFunc
	requestIgnorer   RequestIgnorerFunc
	recovery         RecoveryFunc
	panicPropagation bool
}

type netHTTPBody struct {
	b []byte
}

// Option sets options for tracing server requests.
type ServerOption func(*apmHandler)

// RequestNameFunc is the type of a function for use in
// WithServerRequestName.
type RequestNameFunc func(*fasthttp.RequestCtx) string

// RequestIgnorerFunc is the type of a function for use in
// WithServerRequestIgnorer.
type RequestIgnorerFunc func(*fasthttp.RequestCtx) bool

// Transaction wraps the APM transaction.
// Implements the `io.Closer` interface to end the transaction automatically,
// due to it will be saved on the RequestCtx.UserValues
// which will end the transaction automatically when the request finish.
type Transaction struct {
	tracer    *apm.Tracer
	tx        *apm.Transaction
	req       http.Request
	httpCtx   *fasthttp.RequestCtx
	body      *apm.BodyCapturer
	httpBody  netHTTPBody
	manualEnd bool
}

// RecoveryFunc is the type of a function for use in WithRecovery.
type RecoveryFunc func(ctx *fasthttp.RequestCtx, tx *Transaction, recovered interface{})
