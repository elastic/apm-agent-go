package apmfasthttp

import (
	"github.com/valyala/fasthttp"
	"go.elastic.co/apm"
)

// NewTxCloser returns a transaction closer.
func NewTxCloser(ctx *fasthttp.RequestCtx, tx *apm.Transaction, bc *apm.BodyCapturer) *TxCloser {
	return &TxCloser{
		ctx: ctx,
		tx:  tx,
		bc:  bc,
	}
}

// Tx returns the underlying APM transaction.
func (c *TxCloser) Tx() *apm.Transaction {
	return c.tx
}

// Tx returns the underlying APM body capturer.
func (c *TxCloser) BodyCapturer() *apm.BodyCapturer {
	return c.bc
}

// Close sets the response context to the APM transaction and
// ends the transaction.
func (c *TxCloser) Close() error {
	if err := setResponseContext(c.ctx, c.tx, c.bc); err != nil {
		return err
	}

	c.tx.End()

	return nil
}
