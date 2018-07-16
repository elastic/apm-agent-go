package apmot

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/module/apmhttp"
)

// spanImpl wraps elasticapm objects to implement the opentracing.Span interface.
type spanImpl struct {
	tracer *tracerImpl

	mu   sync.Mutex
	tx   *elasticapm.Transaction
	span *elasticapm.Span
	tags opentracing.Tags
	ctx  SpanContext
}

// SetOperationName sets or changes the operation name.
func (s *spanImpl) SetOperationName(operationName string) opentracing.Span {
	if s.tx != nil {
		s.tx.Name = operationName
	} else {
		s.span.Name = operationName
	}
	return s
}

// SetTag adds or changes a tag.
func (s *spanImpl) SetTag(key string, value interface{}) opentracing.Span {
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
func (s *spanImpl) Finish() {
	s.FinishWithOptions(opentracing.FinishOptions{})
}

// FinishWithOptions is like Finish, but provides explicit control over the
// end timestamp and log data.
func (s *spanImpl) FinishWithOptions(opts opentracing.FinishOptions) {
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
func (s *spanImpl) Tracer() opentracing.Tracer {
	return s.tracer
}

// Context returns the span's current context.
//
// It is valid to call Context after calling Finish or FinishWithOptions.
// The resulting context is also valid after the span is finished.
func (s *spanImpl) Context() opentracing.SpanContext {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ctx
}

// SetBaggageItem sets a key:value pair on this Span and its SpanContext
// that also propagates to descendants of this Span.
func (s *spanImpl) SetBaggageItem(key, val string) opentracing.Span {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx = s.ctx.WithBaggageItem(key, val)
	return s
}

// BaggageItem returns tthe value for a baggage item given its key.
// If there is no baggage item for the given key, the empty string
// is returned.
func (s *spanImpl) BaggageItem(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ctx.baggage[key]
}

func (*spanImpl) LogKV(keyValues ...interface{}) {
	// No-op.
}

func (*spanImpl) LogFields(fields ...log.Field) {
	// No-op.
}

func (*spanImpl) LogEvent(event string) {
	// No-op.
}

func (*spanImpl) LogEventWithPayload(event string, payload interface{}) {
	// No-op.
}

func (*spanImpl) Log(ld opentracing.LogData) {
	// No-op.
}

func (s *spanImpl) setSpanContext() {
	var dbContext elasticapm.DatabaseSpanContext
	var haveDBContext bool
	s.span.Type = "unknown" // fallback
	for k, v := range s.tags {
		switch k {
		case "component":
			s.span.Type = fmt.Sprint(v)
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
		}
	}
	if haveDBContext {
		dbType := "sql"
		if dbContext.Type != "" {
			dbType = dbContext.Type
		}
		s.tx.Type = fmt.Sprintf("db.%s.query", dbType)
		s.span.Context.SetDatabase(dbContext)
	}
}

func (s *spanImpl) setTransactionContext() {
	var (
		otCustom   map[string]interface{}
		httpMethod string
		httpURL    string
	)
	for k, v := range s.tags {
		var shouldSetCustom bool
		switch k {
		case "component":
			s.tx.Type = fmt.Sprint(v)
		case "http.method":
			httpMethod = fmt.Sprint(v)
		case "http.status_code":
			if code, ok := v.(uint64); ok {
				s.tx.Result = apmhttp.StatusCodeResult(int(code))
				s.tx.Context.SetHTTPStatusCode(int(code))
			} else {
				// ???
				s.tx.Result = fmt.Sprint(v)
			}
		case "http.url":
			httpURL = fmt.Sprint(v)
		default:
			shouldSetCustom = true
		}
		if shouldSetCustom {
			if otCustom == nil {
				otCustom = make(map[string]interface{})
				s.tx.Context.SetCustom("ot", otCustom)
			}
			setCustom(otCustom, k, v)
		}
	}

	if httpURL != "" || s.tx.Type == "" {
		s.tx.Type = "request"
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

func setCustom(out map[string]interface{}, k string, v interface{}) {
	for {
		pos := strings.IndexRune(k, '.')
		if pos == -1 {
			break
		}
		prefix := k[:pos]
		outk, _ := out[k].(map[string]interface{})
		if outk == nil {
			outk = make(map[string]interface{})
			out[prefix] = outk
		}
		k = k[pos+1:]
		out = outk
	}
	out[k] = v
}
