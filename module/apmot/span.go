package apmot

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"

	opentracing "github.com/opentracing/opentracing-go"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

// otSpan wraps apm objects to implement the opentracing.Span interface.
type otSpan struct {
	tracer *otTracer
	unsupportedSpanMethods

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
		s.setSpanContext()
		s.span.End()
	} else {
		s.setTransactionContext()
		s.ctx.mu.Lock()
		tx := s.ctx.tx
		s.ctx.tx = nil
		s.ctx.mu.Unlock()
		tx.End()
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

// SetBaggageItem is a no-op; we do not support baggage.
func (s *otSpan) SetBaggageItem(key, val string) opentracing.Span {
	// We do not support baggage.
	return s
}

func (s *otSpan) setSpanContext() {
	var (
		dbContext       apm.DatabaseSpanContext
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

		default:
			s.span.Context.SetTag(k, fmt.Sprint(v))
		}
	}
	switch {
	case haveHTTPContext:
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
	case haveDBContext:
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
			s.ctx.tx.Type = fmt.Sprint(v)
		case "result":
			s.ctx.tx.Result = fmt.Sprint(v)
		case "user.id":
			s.ctx.tx.Context.SetUserID(fmt.Sprint(v))
		case "user.email":
			s.ctx.tx.Context.SetUserEmail(fmt.Sprint(v))
		case "user.username":
			s.ctx.tx.Context.SetUsername(fmt.Sprint(v))

		default:
			s.ctx.tx.Context.SetTag(k, fmt.Sprint(v))
		}
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
