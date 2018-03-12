package trace

import (
	"sync"
	"time"

	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/stacktrace"
)

// StartTransaction returns a new Transaction with the specified
// name and type, and with the start time set to the current time.
func (t *Tracer) StartTransaction(name, type_ string) *Transaction {
	tx := t.newTransaction(name, type_)
	tx.Timestamp = time.Now()
	return tx
}

// newTransaction returns a new Transaction with the specified
// name and type, and sampling applied.
func (t *Tracer) newTransaction(name, type_ string) *Transaction {
	tx, _ := t.transactionPool.Get().(*Transaction)
	if tx == nil {
		tx = &Transaction{tracer: t}
	}
	tx.Name = name
	tx.Type = type_

	// Take a snapshot of the max spans config to ensure
	// that once the maximum is reached, all future span
	// creations are dropped.
	t.maxSpansMu.RLock()
	tx.maxSpans = t.maxSpans
	t.maxSpansMu.RUnlock()

	t.samplerMu.RLock()
	sampler := t.sampler
	t.samplerMu.RUnlock()
	tx.sampled = true
	if sampler != nil && !sampler.Sample(tx) {
		tx.sampled = false
		tx.Transaction.Sampled = &tx.sampled
	}
	return tx
}

// Transaction describes an event occurring in the monitored service.
//
// The ID, Spans, and SpanCount fields should not be modified
// directly. ID will be set by the Tracer when the transaction is
// flushed; the Span and SpanCount fields will be updated by the
// StartSpan method.
//
// Multiple goroutines should not attempt to update the Transaction
// fields concurrently, but may concurrently invoke any methods.
type Transaction struct {
	model.Transaction

	tracer   *Tracer
	sampled  bool
	maxSpans int

	mu           sync.Mutex
	tags         []tag
	spans        []*Span
	spansDropped int
}

type tag struct {
	key, value string
}

func (tx *Transaction) setID() {
	if tx.Transaction.ID != "" {
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
	tx.Transaction.ID = transactionID
}

func (tx *Transaction) setContext(setter stacktrace.ContextSetter, pre, post int) error {
	for _, s := range tx.Spans {
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
	tags := tx.tags[:0]
	spans := tx.spans[:0]
	modelSpans := tx.Spans[:0]
	tracer := tx.tracer
	*tx = Transaction{}
	tx.tags = tags
	tx.tracer = tracer
	tx.spans = spans
	tx.Spans = modelSpans
}

// Sampled reports whether or not the transaction is sampled.
func (tx *Transaction) Sampled() bool {
	return tx.sampled
}

// SetTag sets a tag on the transaction, returning true if
// the tag is added to the transaction, false otherwise.
// The tag will not be added to a non-sampled transaction,
// or if the tag key is invalid (contains '.', '*', or '"').
func (tx *Transaction) SetTag(key, value string) bool {
	if !tx.Sampled() || !validTagKey(key) {
		return false
	}
	tx.mu.Lock()
	tx.tags = append(tx.tags, tag{key, value})
	tx.mu.Unlock()
	return true
}

// Done sets the transaction's duration to the specified value, and
// enqueues it for sending to the Elastic APM server. The Transaction
// must not be used after this.
//
// If the duration specified is negative, then Done will set the
// duration to "time.Since(tx.Timestamp)" instead.
func (tx *Transaction) Done(d time.Duration) {
	if d < 0 {
		d = time.Since(tx.Timestamp)
	}
	tx.Duration = d

	tx.mu.Lock()
	spans := tx.spans[:len(tx.spans)]
	tags := tx.tags[:len(tx.tags)]
	tx.mu.Unlock()
	if len(spans) != 0 {
		tx.Spans = make([]*model.Span, len(spans))
		for i, s := range spans {
			s.truncate(d)
			tx.Spans[i] = &s.Span
		}
	}
	if tx.spansDropped > 0 {
		tx.SpanCount = &model.SpanCount{
			Dropped: &model.SpanCountDropped{
				Total: tx.spansDropped,
			},
		}
	}
	if len(tags) > 0 {
		if tx.Context == nil {
			tx.Context = &model.Context{}
		}
		if tx.Context.Tags == nil {
			tx.Context.Tags = make(map[string]string)
		}
		for _, tag := range tags {
			tx.Context.Tags[tag.key] = tag.value
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
// transaction's timestamp. The span's ID will be set.
//
// If the transaction is not being sampled, then StartSpan will
// return nil.
//
// If the transaction is sampled, then the span's ID will be set,
// and its stacktrace will be set if the tracer is configured
// accordingly.
func (tx *Transaction) StartSpan(name, type_ string, parent *Span) *Span {
	if !tx.Sampled() {
		return nil
	}

	start := time.Since(tx.Timestamp)
	span, _ := tx.tracer.spanPool.Get().(*Span)
	if span == nil {
		span = &Span{}
	}
	span.tx = tx
	span.Name = name
	span.Type = type_
	span.Start = start

	tx.mu.Lock()
	if tx.maxSpans > 0 && len(tx.spans) >= tx.maxSpans {
		span.dropped = true
		tx.spansDropped++
	} else {
		if parent != nil {
			span.Parent = parent.ID
		}
		spanID := int64(len(tx.spans))
		span.ID = &spanID
		tx.spans = append(tx.spans, span)
	}
	tx.mu.Unlock()
	return span
}

// Span describes an operation within a transaction.
type Span struct {
	model.Span
	tx      *Transaction
	dropped bool

	mu        sync.Mutex
	done      bool
	truncated bool
}

func (s *Span) reset() {
	stacktrace := s.Span.Stacktrace[:0]
	*s = Span{}
	s.Span.Stacktrace = stacktrace
}

// SetStacktrace sets the stacktrace for the span,
// skipping the first skip number of frames,
// excluding the SetStacktrace function.
//
// If the span is dropped, this method is a no-op.
func (s *Span) SetStacktrace(skip int) {
	if s.Dropped() {
		return
	}
	// TODO(axw) consider using an LRU cache for
	// the stacktrace frames. We can key on the
	// top of the stack.
	s.Stacktrace = stacktrace.Stacktrace(skip+1, -1)
}

// Dropped indicates whether or not the span is dropped, meaning it
// will not be included in the transaction. Spans are dropped when
// the configurable limit is reached.
func (s *Span) Dropped() bool {
	return s.dropped
}

// Done sets the span's duration to the specified value. The Span
// must not be used after this.
//
// If the duration specified is negative, then Done will set the
// duration to "time.Since(tx.Timestamp.Add(s.Start))" instead.
//
// If the span is dropped, this method is a no-op.
func (s *Span) Done(d time.Duration) {
	if s.Dropped() {
		return
	}
	if d < 0 {
		start := s.tx.Timestamp.Add(s.Start)
		d = time.Since(start)
	}
	s.mu.Lock()
	if !s.truncated {
		s.done = true
		s.Duration = d
	}
	s.mu.Unlock()
}

func (s *Span) truncate(d time.Duration) {
	s.mu.Lock()
	if !s.done {
		s.truncated = true
		s.Type += ".truncated"
		s.Duration = d
	}
	s.mu.Unlock()
}
