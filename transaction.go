package elasticapm

import (
	"sync"
	"time"

	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/stacktrace"
)

// StartTransaction returns a new Transaction with the specified
// name and type, and with the start time set to the current time.
func (t *Tracer) StartTransaction(name, transactionType string, opts ...TransactionOption) *Transaction {
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
	for _, o := range opts {
		o(tx)
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
	tx.Timestamp = time.Now()
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

// TransactionOption sets options when starting a transaction.
type TransactionOption func(*Transaction)
