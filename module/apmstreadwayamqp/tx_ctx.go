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

package apmstreadwayamqp // import "go.elastic.co/apm/module/apmstreadwayamqp/v2"

import (
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
func ExtractTraceContext(del amqp.Delivery) (apm.TraceContext, bool) {
	txCtx, ok := getMessageTraceparent(del.Headers, w3cTraceparentHeader)
	if !ok {
		txCtx, ok = getMessageTraceparent(del.Headers, elasticTraceparentHeader)
	}

	if ok {
		txCtx.State, _ = getMessageTracestate(del.Headers, tracestateHeader)
	}
	return txCtx, ok
}

func getMessageTraceparent(headers map[string]interface{}, header string) (apm.TraceContext, bool) {
	headerValue := getHeaderValueAsStringIfPresent(headers, header)
	if len(headerValue) == 0 {
		return apm.TraceContext{}, false
	}
	if trP, err := apmhttp.ParseTraceparentHeader(headerValue); err == nil {
		return trP, true
	}
	return apm.TraceContext{}, false
}

func getMessageTracestate(headers map[string]interface{}, header string) (apm.TraceState, bool) {
	headerValue := getHeaderValueAsStringIfPresent(headers, header)

	if len(headerValue) == 0 {
		return apm.TraceState{}, false
	}
	if trP, err := apmhttp.ParseTracestateHeader(headerValue); err == nil {
		return trP, true
	}
	return apm.TraceState{}, false
}

func getHeaderValueAsStringIfPresent(headers map[string]interface{}, header string) string {
	for h, val := range headers {
		if hv, ok := val.(string); ok && strings.EqualFold(header, h) {
			return hv
		}
	}
	return ""
}
