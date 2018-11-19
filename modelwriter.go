package apm

import (
	"go.elastic.co/apm/internal/ringbuffer"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/stacktrace"
	"go.elastic.co/fastjson"
)

const (
	transactionBlockTag ringbuffer.BlockTag = iota + 1
	spanBlockTag
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
	modelStacktrace []model.StacktraceFrame
}

// writeTransaction encodes tx as JSON to the buffer, and then resets tx.
func (w *modelWriter) writeTransaction(tx *TransactionData) {
	var modelTx model.Transaction
	w.buildModelTransaction(&modelTx, tx)
	w.json.RawString(`{"transaction":`)
	modelTx.MarshalFastJSON(&w.json)
	w.json.RawByte('}')
	w.buffer.WriteBlock(w.json.Bytes(), transactionBlockTag)
	w.json.Reset()
	tx.reset()
}

// writeSpan encodes s as JSON to the buffer, and then resets s.
func (w *modelWriter) writeSpan(s *SpanData) {
	var modelSpan model.Span
	w.buildModelSpan(&modelSpan, s)
	w.json.RawString(`{"span":`)
	modelSpan.MarshalFastJSON(&w.json)
	w.json.RawByte('}')
	w.buffer.WriteBlock(w.json.Bytes(), spanBlockTag)
	w.json.Reset()
	s.reset()
}

// writeError encodes e as JSON to the buffer, and then resets e.
func (w *modelWriter) writeError(e *ErrorData) {
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
		w.json.RawString(`{"metricset":`)
		m.MarshalFastJSON(&w.json)
		w.json.RawByte('}')
		w.buffer.WriteBlock(w.json.Bytes(), metricsBlockTag)
		w.json.Reset()
	}
	m.reset()
}

func (w *modelWriter) buildModelTransaction(out *model.Transaction, tx *TransactionData) {
	out.ID = model.SpanID(tx.traceContext.Span)
	out.TraceID = model.TraceID(tx.traceContext.Trace)
	out.ParentID = model.SpanID(tx.parentSpan)

	out.Name = truncateString(tx.Name)
	out.Type = truncateString(tx.Type)
	out.Result = truncateString(tx.Result)
	out.Timestamp = model.Time(tx.timestamp.UTC())
	out.Duration = tx.Duration.Seconds() * 1000
	out.SpanCount.Started = tx.spansCreated
	out.SpanCount.Dropped = tx.spansDropped

	if !tx.traceContext.Options.Recorded() {
		out.Sampled = &notSampled
	}

	out.Context = tx.Context.build()
	if len(w.cfg.sanitizedFieldNames) != 0 && out.Context != nil {
		if out.Context.Request != nil {
			sanitizeRequest(out.Context.Request, w.cfg.sanitizedFieldNames)
		}
		if out.Context.Response != nil {
			sanitizeResponse(out.Context.Response, w.cfg.sanitizedFieldNames)
		}
	}
}

func (w *modelWriter) buildModelSpan(out *model.Span, span *SpanData) {
	w.modelStacktrace = w.modelStacktrace[:0]
	out.ID = model.SpanID(span.traceContext.Span)
	out.TraceID = model.TraceID(span.traceContext.Trace)
	out.ParentID = model.SpanID(span.parentID)
	out.TransactionID = model.SpanID(span.transactionID)

	out.Name = truncateString(span.Name)
	out.Type = truncateString(span.Type)
	out.Timestamp = model.Time(span.timestamp.UTC())
	out.Duration = span.Duration.Seconds() * 1000
	out.Context = span.Context.build()

	w.modelStacktrace = appendModelStacktraceFrames(w.modelStacktrace, span.stacktrace)
	out.Stacktrace = w.modelStacktrace
	w.setStacktraceContext(out.Stacktrace)
}

func (w *modelWriter) buildModelError(e *ErrorData) {
	// TODO(axw) move the model type outside of Error
	e.model.ID = model.TraceID(e.ID)
	e.model.TraceID = model.TraceID(e.TraceID)
	e.model.ParentID = model.SpanID(e.ParentID)
	e.model.TransactionID = model.SpanID(e.TransactionID)
	e.model.Timestamp = model.Time(e.Timestamp.UTC())
	e.model.Context = e.Context.build()
	e.model.Exception.Handled = e.Handled

	e.setStacktrace()
	e.setCulprit()
	w.setStacktraceContext(e.modelStacktrace)
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
