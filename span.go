package elasticapm

import (
	"sync"
	"time"

	"github.com/elastic/apm-agent-go/model"
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
// StartSpan always returns a non-nil Span. Its Done method must
// be called when the span completes.
func (tx *Transaction) StartSpan(name, spanType string, parent *Span) *Span {
	if tx == nil || !tx.Sampled() {
		return newDroppedSpan()
	}

	var span *Span
	tx.mu.Lock()
	if tx.maxSpans > 0 && len(tx.spans) >= tx.maxSpans {
		tx.model.SpanCount.Dropped.Total++
		tx.mu.Unlock()
		return newDroppedSpan()
	}
	span, _ = tx.tracer.spanPool.Get().(*Span)
	if span == nil {
		span = &Span{}
	}
	span.tx = tx
	span.id = int64(len(tx.spans))
	tx.spans = append(tx.spans, span)
	tx.mu.Unlock()

	span.model.Name = truncateString(name)
	span.model.Type = truncateString(spanType)
	span.Start = time.Now()
	if parent != nil {
		span.model.Parent = parent.model.ID
	}
	return span
}

// Span describes an operation within a transaction.
type Span struct {
	tx      *Transaction // nil if span is dropped
	id      int64
	Start   time.Time
	Context SpanContext

	mu         sync.Mutex
	model      model.Span
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
		model: model.Span{
			Stacktrace: s.model.Stacktrace[:0],
		},
		Context:    s.Context,
		stacktrace: s.stacktrace[:0],
	}
	s.Context.reset()
}

// SetStacktrace sets the stacktrace for the span,
// skipping the first skip number of frames,
// excluding the SetStacktrace function.
func (s *Span) SetStacktrace(skip int) {
	if s.Dropped() {
		return
	}
	s.stacktrace = stacktrace.AppendStacktrace(s.stacktrace[:0], skip+1, -1)
	s.model.Stacktrace = appendModelStacktraceFrames(s.model.Stacktrace[:0], s.stacktrace)
}

// Dropped indicates whether or not the span is dropped, meaning it will not
// be included in any transaction. Spans are dropped by Transaction.StartSpan
// if the transaction is nil, non-sampled, or the transaction's max spans
// limit has been reached.
//
// Dropped may be used to avoid any expensive computation required to set
// the span's context.
func (s *Span) Dropped() bool {
	return s.tx == nil
}

// Done sets the span's duration to the specified value. The Span
// must not be used after this.
//
// If the duration specified is negative, then Done will set the
// duration to "time.Since(s.Start)" instead.
func (s *Span) Done(d time.Duration) {
	if s.Dropped() {
		droppedSpanPool.Put(s)
		return
	}
	if d < 0 {
		d = time.Since(s.Start)
	}
	s.mu.Lock()
	if s.model.ID == nil {
		s.model.ID = &s.id
		s.model.Duration = d.Seconds() * 1000
		if s.model.Stacktrace == nil && d >= s.tx.spanFramesMinDuration {
			s.SetStacktrace(1)
		}
	}
	s.mu.Unlock()
}

func (s *Span) finalize(end time.Time) {
	s.model.Start = s.Start.Sub(s.tx.Timestamp).Seconds() * 1000
	s.model.Context = s.Context.build()

	s.mu.Lock()
	if s.model.ID == nil {
		// s.Done was never called, so mark it as truncated and
		// truncate its duration to the end of the transaction.
		s.model.ID = &s.id
		s.model.Type += ".truncated"
		s.model.Duration = end.Sub(s.Start).Seconds() * 1000
	}
	s.mu.Unlock()
}
