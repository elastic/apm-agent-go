package apm

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"sync"
	"time"

	"go.elastic.co/apm/stacktrace"
)

// droppedSpanPool holds *Spans which are used when the span
// is created for a nil or non-sampled Transaction, or one
// whose max spans limit has been reached.
var droppedSpanPool sync.Pool

// StartSpan starts and returns a new Span within the transaction,
// with the specified name, type, and optional parent span, and
// with the start time set to the current time.
//
// StartSpan always returns a non-nil Span. Its End method must
// be called when the span completes.
//
// StartSpan is equivalent to calling StartSpanOptions with
// SpanOptions.Parent set to the trace context of parent if
// parent is non-nil.
func (tx *Transaction) StartSpan(name, spanType string, parent *Span) *Span {
	var parentTraceContext TraceContext
	if parent != nil {
		parentTraceContext = parent.TraceContext()
	}
	return tx.StartSpanOptions(name, spanType, SpanOptions{
		Parent: parentTraceContext,
	})
}

// StartSpanOptions starts and returns a new Span within the transaction,
// with the specified name, type, and options.
//
// StartSpan always returns a non-nil Span. Its End method must
// be called when the span completes.
func (tx *Transaction) StartSpanOptions(name, spanType string, opts SpanOptions) *Span {
	if tx == nil || !tx.Sampled() {
		return newDroppedSpan()
	}
	tx.mu.Lock()
	if tx.maxSpans > 0 && tx.spansCreated >= tx.maxSpans {
		tx.spansDropped++
		tx.mu.Unlock()
		return newDroppedSpan()
	}
	transactionID := tx.traceContext.Span
	if opts.Parent == (TraceContext{}) {
		opts.Parent = tx.traceContext
	}
	// Calculate the span time relative to the transaction timestamp so
	// that wall-clock adjustments occurring after the transaction start
	// don't affect the span timestamp.
	if opts.Start.IsZero() {
		opts.Start = tx.timestamp.Add(time.Since(tx.timestamp))
	} else {
		opts.Start = tx.timestamp.Add(opts.Start.Sub(tx.timestamp))
	}
	span := tx.tracer.startSpan(name, spanType, transactionID, opts)
	binary.LittleEndian.PutUint64(span.traceContext.Span[:], tx.rand.Uint64())
	span.stackFramesMinDuration = tx.spanFramesMinDuration
	tx.spansCreated++
	tx.mu.Unlock()
	return span
}

// StartSpan returns a new Span with the specified name, type, transaction ID,
// and options. The parent transaction context and transaction IDs must have
// valid, non-zero values, or else the span will be dropped.
//
// In most cases, you should use Transaction.StartSpan or Transaction.StartSpanOptions.
// This method is provided for corner-cases, such as starting a span after the
// containing transaction's End method has been called. Spans created in this
// way will not have the "max spans" configuration applied, nor will they be
// considered in any transaction's span count.
func (t *Tracer) StartSpan(name, spanType string, transactionID SpanID, opts SpanOptions) *Span {
	if opts.Parent.Trace.Validate() != nil || opts.Parent.Span.Validate() != nil || transactionID.Validate() != nil {
		return newDroppedSpan()
	}
	if !opts.Parent.Options.MaybeRecorded() {
		return newDroppedSpan()
	}
	var spanID SpanID
	if _, err := cryptorand.Read(spanID[:]); err != nil {
		return newDroppedSpan()
	}
	if opts.Start.IsZero() {
		opts.Start = time.Now()
	}
	span := t.startSpan(name, spanType, transactionID, opts)
	span.traceContext.Span = spanID
	t.spanFramesMinDurationMu.RLock()
	span.stackFramesMinDuration = t.spanFramesMinDuration
	t.spanFramesMinDurationMu.RUnlock()
	return span
}

func (t *Tracer) startSpan(name, spanType string, transactionID SpanID, opts SpanOptions) *Span {
	span, _ := t.spanPool.Get().(*Span)
	if span == nil {
		span = &Span{
			tracer:   t,
			Duration: -1,
		}
	}
	span.Name = name
	span.Type = spanType
	span.traceContext = opts.Parent
	span.parentID = opts.Parent.Span
	span.transactionID = transactionID
	span.timestamp = opts.Start
	return span
}

// Span describes an operation within a transaction.
type Span struct {
	tracer                 *Tracer // nil if span is dropped
	traceContext           TraceContext
	parentID               SpanID
	transactionID          SpanID
	stackFramesMinDuration time.Duration
	timestamp              time.Time

	Name     string
	Type     string
	Duration time.Duration
	Context  SpanContext

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

// TraceContext returns the span's TraceContext.
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
// since the span's start time.
func (s *Span) End() {
	if s.Dropped() {
		droppedSpanPool.Put(s)
		return
	}
	if s.Duration < 0 {
		s.Duration = time.Since(s.timestamp)
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

// SpanOptions holds options for Transaction.StartSpanOptions and Tracer.StartSpan.
type SpanOptions struct {
	// Parent, if non-zero, holds the trace context of the parent span.
	Parent TraceContext

	// Start is the start time of the span. If this has the zero value,
	// time.Now() will be used instead.
	//
	// When a span is created using Transaction.StartSpanOptions, the
	// span timestamp is internally calculated relative to the transaction
	// timestamp.
	//
	// When Tracer.StartSpan is used, this timestamp should be pre-calculated
	// as relative from the transaction start time, i.e. by calculating the
	// time elapsed since the transaction started, and adding that to the
	// transaction timestamp. Calculating the timstamp in this way will ensure
	// monotonicity of events within a transaction.
	Start time.Time
}
