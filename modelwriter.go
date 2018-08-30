package elasticapm

import (
	"github.com/elastic/apm-agent-go/internal/fastjson"
	"github.com/elastic/apm-agent-go/internal/ringbuffer"
	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/stacktrace"
)

const (
	transactionBlockTag ringbuffer.BlockTag = iota + 1
	errorBlockTag
	metricsBlockTag
)

// notSampled is used as the pointee for the model.Transaction.Sampled field
// of non-sampled transactions.
var notSampled = false

type modelWriter struct {
	buffer          *ringbuffer.Buffer
	cfg             *tracerConfig
	stats           *TracerStats
	json            fastjson.Writer
	modelSpans      []model.Span
	modelStacktrace []model.StacktraceFrame
}

// writeTransaction encodes tx as JSON to the buffer, and then resets tx.
func (w *modelWriter) writeTransaction(tx *Transaction) {
	var modelTx model.Transaction
	w.buildModelTransaction(&modelTx, tx)
	w.json.RawString(`{"transaction":`)
	modelTx.MarshalFastJSON(&w.json)
	w.json.RawByte('}')
	w.buffer.WriteBlock(w.json.Bytes(), transactionBlockTag)
	w.json.Reset()
	tx.reset()
}

// writeError encodes e as JSON to the buffer, and then resets e.
func (w *modelWriter) writeError(e *Error) {
	w.buildModelError(e)
	w.json.RawString(`{"error":`)
	e.model.MarshalFastJSON(&w.json)
	w.json.RawByte('}')
	w.buffer.WriteBlock(w.json.Bytes(), errorBlockTag)
	w.json.Reset()
	e.reset()
}

// writeMetrics encodes m as JSON to the buffer, and then resets m.
func (w *modelWriter) writeMetrics(m *Metrics) {
	for _, m := range m.metrics {
		w.json.RawString(`{"metrics":`)
		m.MarshalFastJSON(&w.json)
		w.json.RawByte('}')
		w.buffer.WriteBlock(w.json.Bytes(), metricsBlockTag)
		w.json.Reset()
	}
	m.reset()
}

func (w *modelWriter) buildModelTransaction(out *model.Transaction, tx *Transaction) {
	// TODO(axw) need to start sending spans independently.
	w.modelSpans = w.modelSpans[:0]
	w.modelStacktrace = w.modelStacktrace[:0]
	out.TraceID = model.TraceID(tx.traceContext.Trace)
	out.ParentID = model.SpanID(tx.parentSpan)
	out.ID = model.SpanID(tx.traceContext.Span)

	out.Name = truncateString(tx.Name)
	out.Type = truncateString(tx.Type)
	out.Result = truncateString(tx.Result)
	out.Timestamp = model.Time(tx.Timestamp.UTC())
	out.Duration = tx.Duration.Seconds() * 1000
	out.SpanCount.Dropped.Total = tx.spansDropped

	if !tx.Sampled() {
		out.Sampled = &notSampled
	}

	out.Context = tx.Context.build()
	if w.cfg.sanitizedFieldNames != nil && out.Context != nil && out.Context.Request != nil {
		sanitizeRequest(out.Context.Request, w.cfg.sanitizedFieldNames)
	}

	spanOffset := len(w.modelSpans)
	for _, span := range tx.spans {
		w.modelSpans = append(w.modelSpans, model.Span{})
		modelSpan := &w.modelSpans[len(w.modelSpans)-1]
		w.buildModelSpan(modelSpan, span)
	}
	out.Spans = w.modelSpans[spanOffset:]
}

func (w *modelWriter) buildModelSpan(out *model.Span, span *Span) {
	out.ID = model.SpanID(span.traceContext.Span)
	out.TraceID = model.TraceID(span.traceContext.Trace)
	out.ParentID = model.SpanID(span.parentID)
	out.TransactionID = model.SpanID(span.transactionID)

	out.Name = truncateString(span.Name)
	out.Type = truncateString(span.Type)
	out.Timestamp = model.Time(span.Timestamp.UTC())
	out.Duration = span.Duration.Seconds() * 1000
	out.Context = span.Context.build()

	stacktraceOffset := len(w.modelStacktrace)
	w.modelStacktrace = appendModelStacktraceFrames(w.modelStacktrace, span.stacktrace)
	out.Stacktrace = w.modelStacktrace[stacktraceOffset:]
	w.setStacktraceContext(out.Stacktrace)
}

func (w *modelWriter) buildModelError(e *Error) {
	// TODO(axw) move the model type outside of Error
	e.model.TraceID = model.TraceID(e.TraceID)
	e.model.ParentID = model.SpanID(e.ParentID)
	e.model.TransactionID = model.SpanID(e.TransactionID)
	e.model.ID = model.SpanID(e.ID)
	e.model.Timestamp = model.Time(e.Timestamp.UTC())
	e.model.Context = e.Context.build()
	e.model.Exception.Handled = e.Handled

	w.setStacktraceContext(e.modelStacktrace)
	e.setStacktrace()
	e.setCulprit()
}

func (w *modelWriter) setStacktraceContext(stack []model.StacktraceFrame) {
	if w.cfg.contextSetter == nil || len(stack) == 0 {
		return
	}
	err := stacktrace.SetContext(w.cfg.contextSetter, stack, w.cfg.preContext, w.cfg.postContext)
	if err != nil {
		if w.cfg.logger != nil {
			w.cfg.logger.Debugf("setting context failed: %v", err)
		}
		w.stats.Errors.SetContext++
	}
}
