package apmfasthttp

import (
	"sync"

	"go.elastic.co/apm"
)

const TxKey = "apmfasthttp_transaction"

var transactionPool = sync.Pool{
	New: func() interface{} {
		return new(Transaction)
	},
}

func acquireTransaction() *Transaction {
	return transactionPool.Get().(*Transaction)
}

func releaseTransaction(tx *Transaction) {
	tx.reset()
	transactionPool.Put(tx)
}

func (tx *Transaction) reset() {
	tx.tracer = nil
	tx.tx = nil
	resetHTTPRequest(&tx.req)
	tx.httpCtx = nil
	tx.body = nil
	tx.httpBody.reset()
	tx.manualEnd = false
}

// AutoEnd defers the tx.End() to call it manually.
//
// So useful, if you need to response in streaming and count too that overtime.
//
// It is enabled by default.
func (tx *Transaction) AutoEnd(v bool) {
	tx.manualEnd = !v
}

// Tx returns the underlying APM transaction.
func (tx *Transaction) Tx() *apm.Transaction {
	return tx.tx
}

// Tx returns the underlying APM Tracer.
func (tx *Transaction) Tracer() *apm.Tracer {
	return tx.tracer
}

// End enqueues tx for sending to the Elastic APM server.
//
// Calling End will set tx's TransactionData field to nil, so callers
// must ensure tx is not updated after End returns.
//
// If the underliying tx.Duration has not been set, End will set it to the elapsed time
// since the transaction's start time.
func (tx *Transaction) End() {
	tx.tx.End()
	releaseTransaction(tx)
}

// Close sets the response context to the APM transaction and
// ends the transaction if the tx.AutoEnd() is called with a false value.
// if not, the transaction will end.
func (tx *Transaction) Close() error {
	if err := setResponseContext(tx); err != nil {
		return err
	}

	if !tx.manualEnd {
		tx.End()
	}

	return nil
}
