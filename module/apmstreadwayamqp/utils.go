package apmstreadwayamqp

import (
	"fmt"
	"go.elastic.co/apm/module/apmhttp/v2"
	"go.elastic.co/apm/v2"
	"strings"
)

var (
	elasticTraceparentHeader = strings.ToLower(apmhttp.ElasticTraceparentHeader)
	w3cTraceparentHeader     = strings.ToLower(apmhttp.W3CTraceparentHeader)
	tracestateHeader         = strings.ToLower(apmhttp.TracestateHeader)
	apmStreadwayAmqpPrefix   = "apmstreadwayamqp_transaction_"
)

func getMessageTraceparent(headers map[string]interface{}, header string) (apm.TraceContext, bool) {
	headerValue, ok := headers[fmt.Sprintf("%s_%s", apmStreadwayAmqpPrefix, header)]
	if !ok || headerValue == nil {
		return apm.TraceContext{}, false
	}
	switch headerValue.(type) {
	case string:
		if trP, err := apmhttp.ParseTraceparentHeader(headerValue.(string)); err == nil {
			return trP, true
		}
	default:
		return apm.TraceContext{}, false
	}
	return apm.TraceContext{}, false
}

func getMessageTracestate(headers map[string]interface{}, header string) (apm.TraceState, bool) {
	headerValue, ok := headers[fmt.Sprintf("%s_%s", apmStreadwayAmqpPrefix, header)]
	if ok {
		if headerValue != nil {
			switch headerValue.(type) {
			case string:
				if trP, err := apmhttp.ParseTracestateHeader(strings.Split(headerValue.(string), ",")...); err == nil {
					return trP, true
				}
			default:
				return apm.TraceState{}, false
			}
		}
	}
	return apm.TraceState{}, false
}

func setMessageTraceparent(tc apm.TraceContext, headers map[string]interface{}) {
	if headers != nil {
		headers[fmt.Sprintf("%s_%s", apmStreadwayAmqpPrefix, w3cTraceparentHeader)] = apmhttp.FormatTraceparentHeader(tc)
	}
}

func setMessageTracestate(tracestate apm.TraceState, headers map[string]interface{}) error {
	if err := tracestate.Validate(); err != nil {
		return err
	}
	if headers != nil {
		headers[fmt.Sprintf("%s_%s", apmStreadwayAmqpPrefix, tracestateHeader)] = tracestate.String()
	}
	return nil
}
