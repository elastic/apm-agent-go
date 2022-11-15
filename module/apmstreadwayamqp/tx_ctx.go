package apmstreadwayamqp

import (
	"context"
	"github.com/streadway/amqp"
	"go.elastic.co/apm/module/apmhttp/v2"
	"go.elastic.co/apm/v2"
	"strings"
)

var (
	elasticTraceparentHeader = strings.ToLower(apmhttp.ElasticTraceparentHeader)
	w3cTraceparentHeader     = strings.ToLower(apmhttp.W3CTraceparentHeader)
	tracestateHeader         = strings.ToLower(apmhttp.TracestateHeader)
)

// InjectTraceContext injects the provided apm.TraceContext in the
// headers of amqp.Publishing
//
// The header injected is W3C Trace-Context header used
// for trace propagation => "Traceparent"
// If the provided msg contains header with this key
// it will be overwritten
func InjectTraceContext(tc apm.TraceContext, msg amqp.Publishing) {
	if msg.Headers != nil {
		msg.Headers[w3cTraceparentHeader] = apmhttp.FormatTraceparentHeader(tc)
		if encoded := tc.State.String(); encoded != "" {
			msg.Headers[tracestateHeader] = encoded
		}
	}
}

// ExtractTraceContext returns apm.TraceContext from the
// trace information stored in the headers.
//
// It's the client's choice how to use the provided apm.TraceContext
func ExtractTraceContext(ctx context.Context, tracer *apm.Tracer, del amqp.Delivery) (apm.TraceContext, error) {
	if err := del.Headers.Validate(); err != nil {
		return apm.TraceContext{}, err
	}

	txCtx, ok := getMessageTraceparent(del.Headers, w3cTraceparentHeader)
	if !ok {
		txCtx, ok = getMessageTraceparent(del.Headers, elasticTraceparentHeader)
	}

	if ok {
		txCtx.State, _ = getMessageTracestate(del.Headers, tracestateHeader)
	}
	return txCtx, nil
}

func getMessageTraceparent(headers map[string]interface{}, header string) (apm.TraceContext, bool) {
	headerValue, ok := headers[header].(string)
	if !ok {
		return apm.TraceContext{}, false
	}
	if trP, err := apmhttp.ParseTraceparentHeader(headerValue); err == nil {
		return trP, true
	}
	return apm.TraceContext{}, false
}

func getMessageTracestate(headers map[string]interface{}, header string) (apm.TraceState, bool) {
	headerValue, ok := headers[header].(string)
	if !ok {
		return apm.TraceState{}, false
	}
	if trP, err := apmhttp.ParseTracestateHeader(strings.Split(headerValue, ",")...); err == nil {
		return trP, true
	}
	return apm.TraceState{}, false
}
