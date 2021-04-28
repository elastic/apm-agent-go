package apmfasthttp

import "io"

func (r *netHTTPBody) Read(p []byte) (int, error) {
	if len(r.b) == 0 {
		return 0, io.EOF
	}

	n := copy(p, r.b)
	r.b = r.b[n:]

	return n, nil
}

func (r *netHTTPBody) Close() error {
	r.reset()

	return nil
}

func (r *netHTTPBody) reset() {
	r.b = r.b[:0]
}
