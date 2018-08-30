package elasticapm

import (
	"encoding/binary"
	"sync"
	"time"

	"github.com/elastic/apm-agent-go/stacktrace"
)

// droppedSpanPool holds *Spans which are used when the span
// is created for a nil or non-sampled Transaction, or one
// whose max spans limit has been reached.
var droppedSpanPool sync.Pool

// StartSpan starts and returns a new Span within the transaction,
// with the specified name, type, and optional parent span, and
// with the start time set to the current time relative to the
// transaction's timestamp.
//
// StartSpan always returns a non-nil Span. Its End method must
// be called when the span completes.
func (tx *Transaction) StartSpan(name, spanType string, parent *Span) *Span {
	if tx == nil || !tx.Sampled() {
		return newDroppedSpan()
	}

	tx.mu.Lock()
	if tx.maxSpans > 0 && tx.spansCreated >= tx.maxSpans {
		tx.spansDropped++
		tx.mu.Unlock()
		return newDroppedSpan()
	}
	span, _ := tx.tracer.spanPool.Get().(*Span)
	if span == nil {
		span = &Span{
			tracer:   tx.tracer,
			Duration: -1,
		}
	}
	var spanID SpanID
	binary.LittleEndian.PutUint64(spanID[:], tx.rand.Uint64())
	// TODO(axw) profile whether it's worthwhile threading and
	// storing spanFramesMinDuration through to the transaction
	// and span, or if we can instead unconditionally capture
	// the stack trace, and make the rendering in the model
	// writer conditional.
	span.stackFramesMinDuration = tx.spanFramesMinDuration
	tx.spansCreated++
	tx.mu.Unlock()

	span.Name = name
	span.Type = spanType
	span.Timestamp = time.Now()
	if parent != nil {
		span.traceContext = parent.traceContext
	} else {
		span.traceContext = tx.traceContext
	}
	span.parentID = span.traceContext.Span
	span.transactionID = tx.traceContext.Span
	span.traceContext.Span = spanID
	return span
}

// Span describes an operation within a transaction.
type Span struct {
	tracer                 *Tracer // nil if span is dropped
	traceContext           TraceContext
	parentID               SpanID
	transactionID          SpanID
	stackFramesMinDuration time.Duration

	Name      string
	Type      string
	Timestamp time.Time
	Duration  time.Duration
	Context   SpanContext

	stacktrace []stacktrace.Frame
}

func newDroppedSpan() *Span {
	span, _ := droppedSpanPool.Get().(*Span)
	if span == nil {
		span = &Span{}
	}
	return span
}

func (s *Span) reset() {
	*s = Span{
		tracer:     s.tracer,
		Context:    s.Context,
		Duration:   -1,
		stacktrace: s.stacktrace[:0],
	}
	s.Context.reset()
	s.tracer.spanPool.Put(s)
}

// TraceContext returns the span's TraceContext: its trace ID, span ID,
// and trace options. The values are undefined if distributed tracing
// is disabled. If the span is dropped, the trace ID and options will
// be zero.
func (s *Span) TraceContext() TraceContext {
	return s.traceContext
}

// SetStacktrace sets the stacktrace for the span,
// skipping the first skip number of frames,
// excluding the SetStacktrace function.
func (s *Span) SetStacktrace(skip int) {
	if s.Dropped() {
		return
	}
	s.stacktrace = stacktrace.AppendStacktrace(s.stacktrace[:0], skip+1, -1)
}

// Dropped indicates whether or not the span is dropped, meaning it will not
// be included in any transaction. Spans are dropped by Transaction.StartSpan
// if the transaction is nil, non-sampled, or the transaction's max spans
// limit has been reached.
//
// Dropped may be used to avoid any expensive computation required to set
// the span's context.
func (s *Span) Dropped() bool {
	return s.tracer == nil
}

// End marks the s as being complete; s must not be used after this.
//
// If s.Duration has not been set, End will set it to the elapsed time
// since s.Timestamp.
func (s *Span) End() {
	if s.Dropped() {
		droppedSpanPool.Put(s)
		return
	}
	if s.Duration < 0 {
		s.Duration = time.Since(s.Timestamp)
	}
	if len(s.stacktrace) == 0 && s.Duration >= s.stackFramesMinDuration {
		s.SetStacktrace(1)
	}
	s.enqueue()
}

func (s *Span) enqueue() {
	select {
	case s.tracer.spans <- s:
	default:
		// Enqueuing a span should never block.
		s.tracer.statsMu.Lock()
		s.tracer.stats.SpansDropped++
		s.tracer.statsMu.Unlock()
		s.reset()
	}
}
