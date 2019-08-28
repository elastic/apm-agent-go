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

package apmot

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

// otSpan wraps apm objects to implement the opentracing.Span interface.
type otSpan struct {
	tracer *otTracer

	mu   sync.Mutex
	span *apm.Span
	tags opentracing.Tags
	ctx  spanContext
}

// Span returns s.span, the underlying apm.Span. This is used to satisfy
// SpanFromContext.
func (s *otSpan) Span() *apm.Span {
	return s.span
}

// SetOperationName sets or changes the operation name.
func (s *otSpan) SetOperationName(operationName string) opentracing.Span {
	if s.span != nil {
		s.span.Name = operationName
	} else {
		s.ctx.tx.Name = operationName
	}
	return s
}

// SetTag adds or changes a tag.
func (s *otSpan) SetTag(key string, value interface{}) opentracing.Span {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tags == nil {
		s.tags = make(opentracing.Tags, 1)
	}
	s.tags[key] = value
	return s
}

// Finish ends the span; this (or FinishWithOptions) must be the last method
// call on the span, except for calls to Context which may be called at any
// time.
func (s *otSpan) Finish() {
	s.FinishWithOptions(opentracing.FinishOptions{})
}

// FinishWithOptions is like Finish, but provides explicit control over the
// end timestamp and log data.
func (s *otSpan) FinishWithOptions(opts opentracing.FinishOptions) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !opts.FinishTime.IsZero() {
		duration := opts.FinishTime.Sub(s.ctx.startTime)
		if s.span != nil {
			s.span.Duration = duration
		} else {
			s.ctx.tx.Duration = duration
		}
	}
	if s.span != nil {
		for _, record := range opts.LogRecords {
			timestamp := record.Timestamp
			if timestamp.IsZero() {
				timestamp = opts.FinishTime
			}
			logFields(s.tracer.tracer, nil, s.span, timestamp, record.Fields)
		}
		s.setSpanContext()
		s.span.End()
	} else {
		s.setTransactionContext()
		for _, record := range opts.LogRecords {
			timestamp := record.Timestamp
			if timestamp.IsZero() {
				timestamp = opts.FinishTime
			}
			logFields(s.tracer.tracer, s.ctx.tx, nil, timestamp, record.Fields)
		}
		s.ctx.tx.End()
	}
}

// Tracer returns the Tracer that created this span.
func (s *otSpan) Tracer() opentracing.Tracer {
	return s.tracer
}

// Context returns the span's current context.
//
// It is valid to call Context after calling Finish or FinishWithOptions.
// The resulting context is also valid after the span is finished.
func (s *otSpan) Context() opentracing.SpanContext {
	return &s.ctx
}

// BaggageItem returns the empty string; we do not support baggage.
func (*otSpan) BaggageItem(key string) string {
	return ""
}

// SetBaggageItem is a no-op; we do not support baggage.
func (s *otSpan) SetBaggageItem(key, val string) opentracing.Span {
	// We do not support baggage.
	return s
}

// getAndDelete returns value of the key and deletes the entry, or "" if key not found
func getAndDelete(tags *map[string]string, key string) string {
	if value, ok := (*tags)[key]; ok {
		delete(*tags, key)
		return value
	}
	return ""
}

// clone returns a copy of tag map with all values stringified
func clone(tags map[string]interface{}) *map[string]string {
	cloned := make(map[string]string, len(tags))
	for k, v := range tags {
		cloned[k] = fmt.Sprint(v)
	}
	return &cloned
}

func (s *otSpan) setSpanContext() {
	tags := clone(s.tags)
	component := getAndDelete(tags, "component")
	dbContext := apm.DatabaseSpanContext{
		Instance:  getAndDelete(tags, "db.instance"),
		Statement: getAndDelete(tags, "db.statement"),
		Type:      getAndDelete(tags, "db.type"),
		User:      getAndDelete(tags, "db.user"),
	}
	httpURL := getAndDelete(tags, "http.url")
	httpMethod := getAndDelete(tags, "http.method")

	// Elastic APM-specific tags:
	s.span.Type = getAndDelete(tags, "type")

	for k, v := range *tags {
		s.span.Context.SetTag(k, v)
	}
	switch {
	case httpURL != "":
		if s.span.Type == "" {
			s.span.Type = "external"
			s.span.Subtype = "http"
		}
		url, err := url.Parse(httpURL)
		if err == nil {
			var req http.Request
			req.ProtoMajor = 1 // Assume HTTP/1.1
			req.ProtoMinor = 1
			req.Method = httpMethod
			req.URL = url
			s.span.Context.SetHTTPRequest(&req)
		}
	case dbContext.Type != "":
		if s.span.Type == "" {
			s.span.Type = "db"
			s.span.Subtype = dbContext.Type
		}
		s.span.Context.SetDatabase(dbContext)
	}
	if s.span.Type == "" {
		s.span.Type = "custom"
		s.span.Subtype = component
	}
}

func (s *otSpan) setTransactionContext() {
	var httpStatusCode = -1
	tags := clone(s.tags)
	component := getAndDelete(tags, "component")
	httpMethod := getAndDelete(tags, "http.method")
	httpStatusCode, _ = strconv.Atoi(getAndDelete(tags, "http.status_code"))
	httpURL := getAndDelete(tags, "http.url")
	isError, _ := strconv.ParseBool(getAndDelete(tags, "error"))

	// Elastic APM-specific tags:
	s.ctx.tx.Type = getAndDelete(tags, "type")
	s.ctx.tx.Result = getAndDelete(tags, "result")
	s.ctx.tx.Context.SetUserID(getAndDelete(tags, "user.id"))
	s.ctx.tx.Context.SetUserEmail(getAndDelete(tags, "user.email"))
	s.ctx.tx.Context.SetUsername(getAndDelete(tags, "user.username"))

	for k, v := range *tags {
		s.ctx.tx.Context.SetTag(k, v)
	}
	if s.ctx.tx.Type == "" {
		if httpURL != "" {
			s.ctx.tx.Type = "request"
		} else if component != "" {
			s.ctx.tx.Type = component
		} else {
			s.ctx.tx.Type = "custom"
		}
	}
	if s.ctx.tx.Result == "" {
		if httpStatusCode != -1 {
			s.ctx.tx.Result = apmhttp.StatusCodeResult(httpStatusCode)
			s.ctx.tx.Context.SetHTTPStatusCode(httpStatusCode)
		} else if isError {
			s.ctx.tx.Result = "error"
		}
	}
	if httpURL != "" {
		uri, err := url.ParseRequestURI(httpURL)
		if err == nil {
			var req http.Request
			req.ProtoMajor = 1 // Assume HTTP/1.1
			req.ProtoMinor = 1
			req.Method = httpMethod
			req.URL = uri
			s.ctx.tx.Context.SetHTTPRequest(&req)
		}
	}
}

// LogKV is part of the opentracing.Span interface.
// We send error events to Elastic APM.
func (s *otSpan) LogKV(keyValues ...interface{}) {
	logKV(s.tracer.tracer, s.ctx.tx, s.span, time.Time{}, keyValues)
}

// LogFields is part of the opentracing.Span interface.
// We send error events to Elastic APM.
func (s *otSpan) LogFields(fields ...log.Field) {
	logFields(s.tracer.tracer, s.ctx.tx, s.span, time.Time{}, fields)
}

// LogEvent is deprecated, and is a no-op.
func (s *otSpan) LogEvent(event string) {
	// Deprecated, no-op.
}

// LogEventWithPayload is deprecated, and is a no-op.
func (s *otSpan) LogEventWithPayload(event string, payload interface{}) {
	// Deprecated, no-op.
}

// Log is deprecated, and is a no-op.
func (s *otSpan) Log(ld opentracing.LogData) {
	// Deprecated, no-op.
}
