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

// StartTransaction returns a new Transaction with the specified
// name and type, and with the start time set to the current time.
func (t *Tracer) StartTransaction(name, transactionType string) *Transaction {
	tx := t.newTransaction(name, transactionType)
	tx.Timestamp = time.Now()
	return tx
}

// newTransaction returns a new Transaction with the specified
// name and type, and sampling applied.
func (t *Tracer) newTransaction(name, transactionType string) *Transaction {
	tx, _ := t.transactionPool.Get().(*Transaction)
	if tx == nil {
		tx = &Transaction{
			tracer: t,
			Context: Context{
				captureBodyMask: CaptureBodyTransactions,
			},
		}
	}
	tx.model.Name = truncateString(name)
	tx.model.Type = truncateString(transactionType)

	t.transactionIgnoreNamesMu.RLock()
	transactionIgnoreNames := t.transactionIgnoreNames
	t.transactionIgnoreNamesMu.RUnlock()
	if transactionIgnoreNames != nil && transactionIgnoreNames.MatchString(name) {
		tx.ignored = true
		return tx
	}

	// Take a snapshot of the max spans config to ensure
	// that once the maximum is reached, all future span
	// creations are dropped.
	t.maxSpansMu.RLock()
	tx.maxSpans = t.maxSpans
	t.maxSpansMu.RUnlock()

	t.spanFramesMinDurationMu.RLock()
	tx.spanFramesMinDuration = t.spanFramesMinDuration
	t.spanFramesMinDurationMu.RUnlock()

	t.samplerMu.RLock()
	sampler := t.sampler
	t.samplerMu.RUnlock()
	tx.sampled = true
	if sampler != nil && !sampler.Sample(tx) {
		tx.sampled = false
		tx.model.Sampled = &tx.sampled
	}
	return tx
}

// Transaction describes an event occurring in the monitored service.
type Transaction struct {
	model     model.Transaction
	Timestamp time.Time
	Context   Context
	Result    string

	tracer                *Tracer
	ignored               bool
	sampled               bool
	maxSpans              int
	spanFramesMinDuration time.Duration

	mu    sync.Mutex
	spans []*Span
}

func (tx *Transaction) GetID() string {
	return tx.model.ID
}

func (tx *Transaction) SetID(id string) {
	tx.model.ID = id
}

func (tx *Transaction) setID() {
	if tx.model.ID != "" {
		return
	}
	transactionID, err := NewUUID()
	if err != nil {
		// We ignore the error from NewUUID, which will
		// only occur if the entropy source fails. In
		// that case, there's nothing we can do. We don't
		// want to panic inside the user's application.
		return
	}
	tx.model.ID = transactionID
}

func (tx *Transaction) setContext(setter stacktrace.ContextSetter, pre, post int) error {
	for _, s := range tx.model.Spans {
		if err := stacktrace.SetContext(setter, s.Stacktrace, pre, post); err != nil {
			return err
		}
	}
	return nil
}

// reset resets the Transaction back to its zero state, so it can be reused
// in the transaction pool.
func (tx *Transaction) reset() {
	for _, s := range tx.spans {
		s.reset()
		tx.tracer.spanPool.Put(s)
	}
	*tx = Transaction{
		model: model.Transaction{
			Spans: tx.model.Spans[:0],
		},
		tracer:  tx.tracer,
		spans:   tx.spans[:0],
		Context: tx.Context,
	}
	tx.Context.reset()
}

// Discard discards a previously started transaction. The Transaction
// must not be used after this.
func (tx *Transaction) Discard() {
	tx.reset()
	tx.tracer.transactionPool.Put(tx)
}

// Ignored reports whether or not the transaction is ignored.
//
// Transactions may be ignored based on the configuration of
// the tracer. Ignored also implies that the transaction is not
// sampled.
func (tx *Transaction) Ignored() bool {
	return tx.ignored
}

// Sampled reports whether or not the transaction is sampled.
func (tx *Transaction) Sampled() bool {
	return tx.sampled
}

// Done sets the transaction's duration to the specified value, and
// enqueues it for sending to the Elastic APM server. The Transaction
// must not be used after this.
//
// If the duration specified is negative, then Done will set the
// duration to "time.Since(tx.Timestamp)" instead.
func (tx *Transaction) Done(d time.Duration) {
	if tx.ignored {
		tx.Discard()
		return
	}
	if d < 0 {
		d = time.Since(tx.Timestamp)
	}
	tx.model.Duration = d.Seconds() * 1000
	tx.model.Timestamp = model.Time(tx.Timestamp.UTC())
	tx.model.Result = truncateString(tx.Result)

	tx.mu.Lock()
	spans := tx.spans[:len(tx.spans)]
	tx.mu.Unlock()
	if len(spans) != 0 {
		tx.model.Spans = make([]*model.Span, len(spans))
		for i, s := range spans {
			s.finalize(tx.Timestamp.Add(d))
			tx.model.Spans[i] = &s.model
		}
	}

	tx.enqueue()
}

func (tx *Transaction) enqueue() {
	select {
	case tx.tracer.transactions <- tx:
	default:
		// Enqueuing a transaction should never block.
		tx.tracer.statsMu.Lock()
		tx.tracer.stats.TransactionsDropped++
		tx.tracer.statsMu.Unlock()
		tx.reset()
		tx.tracer.transactionPool.Put(tx)
	}
}

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
