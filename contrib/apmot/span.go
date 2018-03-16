package apmot

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/model"
)

// spanImpl wraps elasticapm objects to implement the opentracing.Span interface.
type spanImpl struct {
	tracer *tracerImpl
	tx     *elasticapm.Transaction
	span   *elasticapm.Span

	mu   sync.Mutex
	tags opentracing.Tags
	ctx  SpanContext
}

func (s *spanImpl) SetOperationName(operationName string) opentracing.Span {
	if s.span != nil {
		s.span.Name = operationName
	} else {
		s.tx.Name = operationName
	}
	return s
}

func (s *spanImpl) SetTag(key string, value interface{}) opentracing.Span {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tags == nil {
		s.tags = make(opentracing.Tags, 1)
	}
	s.tags[key] = value
	return s
}

func (s *spanImpl) Finish() {
	s.FinishWithOptions(opentracing.FinishOptions{})
}

func (s *spanImpl) FinishWithOptions(opts opentracing.FinishOptions) {
	var duration time.Duration = -1
	if !opts.FinishTime.IsZero() {
		if s.span != nil {
			duration = opts.FinishTime.Sub(s.span.Start)
		} else {
			duration = opts.FinishTime.Sub(s.tx.Timestamp)
		}
	}
	if s.span != nil {
		s.setSpanContext()
		s.span.Done(duration)
	} else {
		s.setTransactionContext()
		s.tx.Done(duration)
	}
}

func (s *spanImpl) Tracer() opentracing.Tracer {
	return s.tracer
}

func (s *spanImpl) Context() opentracing.SpanContext {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ctx
}

func (s *spanImpl) SetBaggageItem(key, val string) opentracing.Span {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx = s.ctx.WithBaggageItem(key, val)
	return s
}

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
	s.mu.Lock()
	defer s.mu.Unlock()
	var dbContext model.DatabaseSpanContext
	var haveDBContext bool
	s.span.Type = "unknown" // fallback
	for k, v := range s.tags {
		switch k {
		case "component":
			s.tx.Type = fmt.Sprint(v)
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
		s.span.Context = &model.SpanContext{
			Database: &dbContext,
		}
	}
}

func (s *spanImpl) setTransactionContext() {
	s.mu.Lock()
	defer s.mu.Unlock()
	var (
		ctx          model.Context
		otCustom     map[string]interface{}
		httpRequest  model.Request
		httpResponse model.Response
		httpURL      string
	)
	for k, v := range s.tags {
		var shouldSetCustom bool
		switch k {
		case "component":
			s.tx.Type = fmt.Sprint(v)
			shouldSetCustom = true
		case "http.method":
			httpRequest.Method = fmt.Sprint(v)
			ctx.Request = &httpRequest
		case "http.status_code":
			s.tx.Result = fmt.Sprint(v)
			if code, ok := v.(uint64); ok {
				httpResponse.StatusCode = int(code)
			}
			ctx.Response = &httpResponse
		case "http.url":
			httpURL = fmt.Sprint(v)
		default:
			shouldSetCustom = true
		}
		if shouldSetCustom {
			if ctx.Custom == nil {
				otCustom = make(map[string]interface{})
				ctx.Custom = map[string]interface{}{"ot": otCustom}
			}
			setCustom(otCustom, k, v)
		}
	}
	if httpURL != "" || ctx.Custom != nil {
		s.tx.Context = &ctx
	}
	if httpURL != "" || s.tx.Type == "" {
		s.tx.Type = "request"
	}
	if httpURL != "" {
		uri, err := url.ParseRequestURI(httpURL)
		if err == nil {
			httpRequest.URL.Protocol = uri.Scheme
			httpRequest.URL.Path = uri.Path
			httpRequest.URL.Search = uri.RawQuery
			httpRequest.URL.Hash = uri.Fragment
			ctx.Request = &httpRequest
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
