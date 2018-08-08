package apmot

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/module/apmhttp"
)

// otSpan wraps elasticapm objects to implement the opentracing.Span interface.
type otSpan struct {
	tracer *otTracer

	mu   sync.Mutex
	tx   *elasticapm.Transaction
	span *elasticapm.Span
	tags opentracing.Tags
	ctx  spanContext
}

// SetOperationName sets or changes the operation name.
func (s *otSpan) SetOperationName(operationName string) opentracing.Span {
	if s.tx != nil {
		s.tx.Name = operationName
	} else {
		s.span.Name = operationName
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
		if s.span != nil {
			s.span.Duration = opts.FinishTime.Sub(s.span.Timestamp)
		} else {
			s.tx.Duration = opts.FinishTime.Sub(s.tx.Timestamp)
		}
	}
	if s.span != nil {
		s.setSpanContext()
		s.span.End()
	} else {
		s.setTransactionContext()
		s.tx.End()
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
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ctx
}

// SetBaggageItem is a no-op; we do not support baggage.
func (s *otSpan) SetBaggageItem(key, val string) opentracing.Span {
	// We do not support baggage.
	return s
}

// BaggageItem returns the empty string; we do not support baggage.
func (s *otSpan) BaggageItem(key string) string {
	return ""
}

func (*otSpan) LogKV(keyValues ...interface{}) {
	// No-op.
}

func (*otSpan) LogFields(fields ...log.Field) {
	// No-op.
}

func (*otSpan) LogEvent(event string) {
	// No-op.
}

func (*otSpan) LogEventWithPayload(event string, payload interface{}) {
	// No-op.
}

func (*otSpan) Log(ld opentracing.LogData) {
	// No-op.
}

func (s *otSpan) setSpanContext() {
	var (
		dbContext       elasticapm.DatabaseSpanContext
		component       string
		httpURL         string
		httpMethod      string
		haveDBContext   bool
		haveHTTPContext bool
	)
	for k, v := range s.tags {
		switch k {
		case "component":
			component = fmt.Sprint(v)
		case "db.instance":
			dbContext.Instance = fmt.Sprint(v)
			haveDBContext = true
		case "db.statement":
			dbContext.Statement = fmt.Sprint(v)
			haveDBContext = true
		case "db.type":
			dbContext.Type = fmt.Sprint(v)
			haveDBContext = true
		case "db.user":
			dbContext.User = fmt.Sprint(v)
			haveDBContext = true
		case "http.url":
			haveHTTPContext = true
			httpURL = fmt.Sprint(v)
		case "http.method":
			haveHTTPContext = true
			httpMethod = fmt.Sprint(v)

		// Elastic APM-specific tags:
		case "type":
			s.span.Type = fmt.Sprint(v)
		}
	}
	switch {
	case haveHTTPContext:
		if s.span.Type == "" {
			s.span.Type = "ext.http"
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
	case haveDBContext:
		if s.span.Type == "" {
			dbType := "sql"
			if dbContext.Type != "" {
				dbType = dbContext.Type
			}
			s.span.Type = fmt.Sprintf("db.%s.query", dbType)
		}
		s.span.Context.SetDatabase(dbContext)
	}
	if s.span.Type == "" {
		s.span.Type = component
		if s.span.Type == "" {
			s.span.Type = "unknown"
		}
	}
}

func (s *otSpan) setTransactionContext() {
	var (
		component      string
		httpMethod     string
		httpStatusCode = -1
		httpURL        string
		isError        bool
	)
	for k, v := range s.tags {
		switch k {
		case "component":
			component = fmt.Sprint(v)
		case "http.method":
			httpMethod = fmt.Sprint(v)
		case "http.status_code":
			if code, ok := v.(uint16); ok {
				httpStatusCode = int(code)
			}
		case "http.url":
			httpURL = fmt.Sprint(v)
		case "error":
			isError, _ = v.(bool)

		// Elastic APM-specific tags:
		case "type":
			s.tx.Type = fmt.Sprint(v)
		case "result":
			s.tx.Result = fmt.Sprint(v)
		case "user.id":
			s.tx.Context.SetUserID(fmt.Sprint(v))
		case "user.email":
			s.tx.Context.SetUserEmail(fmt.Sprint(v))
		case "user.username":
			s.tx.Context.SetUsername(fmt.Sprint(v))

		default:
			s.tx.Context.SetTag(k, fmt.Sprint(v))
		}
	}
	if s.tx.Type == "" {
		if httpURL != "" {
			s.tx.Type = "request"
		} else if component != "" {
			s.tx.Type = component
		} else {
			s.tx.Type = "unknown"
		}
	}
	if s.tx.Result == "" {
		if httpStatusCode != -1 {
			s.tx.Result = apmhttp.StatusCodeResult(httpStatusCode)
			s.tx.Context.SetHTTPStatusCode(httpStatusCode)
		} else if isError {
			s.tx.Result = "error"
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
			s.tx.Context.SetHTTPRequest(&req)
		}
	}
}
