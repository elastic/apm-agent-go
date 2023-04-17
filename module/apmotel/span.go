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

//go:build go1.18
// +build go1.18

package apmotel // import "go.elastic.co/apm/module/apmotel/v2"

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"runtime"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	"go.elastic.co/apm/module/apmhttp/v2"
	"go.elastic.co/apm/v2"
)

type event struct {
	Name       string
	Attributes []attribute.KeyValue
	Time       time.Time
}

type status struct {
	Code        codes.Code
	Description string
}

type span struct {
	mu sync.Mutex

	provider *tracerProvider

	startTime   time.Time
	attributes  []attribute.KeyValue
	events      []event
	spanContext trace.SpanContext
	status      status
	spanKind    trace.SpanKind

	tx   *apm.Transaction
	span *apm.Span
}

func (s *span) End(options ...trace.SpanEndOption) {
	s.mu.Lock()
	defer s.mu.Unlock()

	config := trace.NewSpanEndConfig(options...)

	if !config.Timestamp().IsZero() {
		duration := config.Timestamp().Sub(s.startTime)
		if s.span != nil {
			s.span.Duration = duration
		} else {
			s.tx.Duration = duration
		}
	}

	if s.span != nil {
		s.setSpanAttributes()
		s.span.End()
		return
	}

	s.setTransactionAttributes()
	s.tx.End()
}

func (s *span) AddEvent(name string, opts ...trace.EventOption) {
	c := trace.NewEventConfig(opts...)
	e := event{Name: name, Attributes: c.Attributes(), Time: c.Timestamp()}

	s.mu.Lock()
	s.events = append(s.events, e)
	s.mu.Unlock()
}

func (s *span) IsRecording() bool {
	if s.span != nil {
		return !s.span.Dropped()
	}

	return s.tx.Sampled()
}

func (s *span) RecordError(err error, opts ...trace.EventOption) {
	opts = append(opts, trace.WithAttributes(
		semconv.ExceptionType(typeStr(err)),
		semconv.ExceptionMessage(err.Error()),
	))

	c := trace.NewEventConfig(opts...)
	if c.StackTrace() {
		opts = append(opts, trace.WithAttributes(
			semconv.ExceptionStacktrace(recordStackTrace()),
		))
	}

	s.AddEvent(semconv.ExceptionEventName, opts...)
}

func (s *span) SpanContext() trace.SpanContext {
	return s.spanContext
}

func (s *span) SetStatus(code codes.Code, description string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status.Code > code {
		return
	}

	status := status{Code: code}
	if code == codes.Error {
		status.Description = description
	}

	s.status = status
}

func (s *span) SetName(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.span != nil {
		s.span.Name = name
	} else {
		s.tx.Name = name
	}
}

func (s *span) SetAttributes(kv ...attribute.KeyValue) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, a := range kv {
		if !a.Valid() {
			// Drop all invalid attributes.
			continue
		}

		s.attributes = append(s.attributes, a)
	}
}

func (s *span) TracerProvider() trace.TracerProvider {
	return s.provider
}

func (s *span) setSpanAttributes() {
	var (
		dbContext       apm.DatabaseSpanContext
		httpURL         string
		httpMethod      string
		httpHost        string
		haveDBContext   bool
		haveHTTPContext bool
		haveHTTPHostTag bool
	)

	agentAttrs := make(map[string]interface{})
	for _, v := range s.attributes {
		agentAttrs[string(v.Key)] = v.Value.AsInterface()

		switch v.Key {
		case "component":
			s.span.Subtype = v.Value.Emit()
		case "db.instance":
			dbContext.Instance = v.Value.Emit()
			haveDBContext = true
		case "db.statement":
			dbContext.Statement = v.Value.Emit()
			haveDBContext = true
		case "db.type":
			dbContext.Type = v.Value.Emit()
			haveDBContext = true
		case "db.user":
			dbContext.User = v.Value.Emit()
			haveDBContext = true
		case "http.url":
			haveHTTPContext = true
			httpURL = v.Value.Emit()
		case "http.method":
			haveHTTPContext = true
			httpMethod = v.Value.Emit()
		case "http.host":
			haveHTTPContext = true
			haveHTTPHostTag = true
			httpHost = v.Value.Emit()
		}
	}

	switch {
	case haveHTTPContext:
		if s.spanKind == trace.SpanKindUnspecified {
			s.span.Type = "external"
			s.span.Subtype = "http"
			s.spanKind = trace.SpanKindClient
		}
		url, err := url.Parse(httpURL)
		if err == nil {
			// handles the case where the url.Host hasn't been set.
			// Tries obtaining the host value from the "http.host" tag.
			// If not found, or if the url.Host has a value, it won't
			// mutate the existing host.
			if url.Host == "" && haveHTTPHostTag {
				url.Host = httpHost
			}
			s.span.Context.SetHTTPRequest(&http.Request{
				ProtoMinor: 1,
				ProtoMajor: 1,
				Method:     httpMethod,
				URL:        url,
			})
		}
	case haveDBContext:
		if s.spanKind == trace.SpanKindUnspecified {
			s.span.Type = "db"
			s.span.Subtype = dbContext.Type
			s.spanKind = trace.SpanKindClient
		}
		s.span.Context.SetDatabase(dbContext)
	}

	s.span.Context.SetOTelAttributes(agentAttrs)
	s.span.Context.SetOTelSpanKind(s.spanKind.String())
}

func (s *span) setTransactionAttributes() {
	var (
		isHTTP      bool
		isRPC       bool
		isMessaging = true

		httpMethod     string
		httpStatusCode = -1
		httpURL        string
	)

	agentAttrs := make(map[string]interface{})
	for _, v := range s.attributes {
		agentAttrs[string(v.Key)] = v.Value.AsInterface()

		switch v.Key {
		case "component":
			s.tx.Type = v.Value.Emit()

		case "http.method":
			httpMethod = v.Value.Emit()
		case "http.status_code":
			if code := v.Value.AsInt64(); code > 0 {
				httpStatusCode = int(code)
			}
		case "http.url":
			httpURL = v.Value.Emit()
			isHTTP = true
		case "http.scheme":
			isHTTP = true

		case "rpc.system":
			isRPC = true

		case "messaging.system":
			isMessaging = true

		case "result":
			s.tx.Result = v.Value.Emit()

		case "user.id":
			s.tx.Context.SetUserID(v.Value.Emit())
		case "user.email":
			s.tx.Context.SetUserEmail(v.Value.Emit())
		case "user.username":
			s.tx.Context.SetUsername(v.Value.Emit())
		}
	}

	if s.tx.Type == "" {
		if s.spanKind == trace.SpanKindServer && (isHTTP || isRPC) {
			s.tx.Type = "request"
		} else if s.spanKind == trace.SpanKindConsumer && isMessaging {
			s.tx.Type = "messaging"
		} else {
			s.tx.Type = "unknown"
		}
	}

	if s.tx.Result == "" && httpStatusCode != -1 {
		s.tx.Result = apmhttp.StatusCodeResult(httpStatusCode)
		s.tx.Context.SetHTTPStatusCode(httpStatusCode)
	}

	if isHTTP {
		if uri, err := url.ParseRequestURI(httpURL); err == nil {
			var req http.Request
			req.ProtoMajor = 1 // Assume HTTP/1.1
			req.ProtoMinor = 1
			req.Method = httpMethod
			req.URL = uri
			s.tx.Context.SetHTTPRequest(&req)
		}
	}

	s.tx.Context.SetOTelAttributes(agentAttrs)
	s.tx.Context.SetOTelSpanKind(s.spanKind.String())
}

func typeStr(i interface{}) string {
	t := reflect.TypeOf(i)
	if t.PkgPath() == "" && t.Name() == "" {
		// Likely a builtin type.
		return t.String()
	}
	return fmt.Sprintf("%s.%s", t.PkgPath(), t.Name())
}

func recordStackTrace() string {
	stackTrace := make([]byte, 2048)
	n := runtime.Stack(stackTrace, false)

	return string(stackTrace[0:n])
}
