package transporttest

import (
	"context"

	"github.com/elastic/apm-agent-go/model"
)

// ChannelTransport implements transport.Transport,
// sending payloads to the provided channels as
// request objects. Once a request object has been
// received, an error should be sent to its Result
// channel to unblock the tracer.
type ChannelTransport struct {
	Transactions chan<- SendTransactionsRequest
	Errors       chan<- SendErrorsRequest
}

type SendTransactionsRequest struct {
	Payload *model.TransactionsPayload
	Result  chan<- error
}

type SendErrorsRequest struct {
	Payload *model.ErrorsPayload
	Result  chan<- error
}

func (c *ChannelTransport) SendTransactions(ctx context.Context, payload *model.TransactionsPayload) error {
	result := make(chan error, 1)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case c.Transactions <- SendTransactionsRequest{payload, result}:
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-result:
			return err
		}
	}
}

func (c *ChannelTransport) SendErrors(ctx context.Context, payload *model.ErrorsPayload) error {
	result := make(chan error, 1)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case c.Errors <- SendErrorsRequest{payload, result}:
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-result:
			return err
		}
	}
}
