package transporttest

import (
	"context"
	"io"
	"io/ioutil"

	"go.elastic.co/apm/transport"
)

// Discard is a transport.Transport which discards
// all streams, and returns no errors.
var Discard transport.Transport = ErrorTransport{}

// ErrorTransport is a transport that returns the stored error
// for each method call.
type ErrorTransport struct {
	Error error
}

// SendStream discards the stream and returns t.Error.
func (t ErrorTransport) SendStream(ctx context.Context, r io.Reader) error {
	errc := make(chan error, 1)
	go func() {
		_, err := io.Copy(ioutil.Discard, r)
		errc <- err
	}()
	select {
	case err := <-errc:
		if err != nil {
			return err
		}
		return t.Error
	case <-ctx.Done():
		return ctx.Err()
	}
}
