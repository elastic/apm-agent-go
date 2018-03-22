package apmhttp

import (
	"net/http"
	"time"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/model"
)

// Handler wraps an http.Handler, reporting a new transaction for each request.
//
// The http.Request's context will be updated with the transaction.
type Handler struct {
	// Handler is the original http.Handler to trace.
	Handler http.Handler

	// Recovery is an optional panic recovery handler. If this is
	// non-nil, panics will be recovered and passed to this function,
	// along with the request and response writer. If Recovery is
	// nil, panics will not be recovered.
	Recovery RecoveryFunc

	// RequestName, if non-nil, will be called by ServeHTTP to obtain
	// the transaction name for the request. If this is nil, the
	// package-level RequestName function will be used.
	RequestName func(*http.Request) string

	// Tracer is an optional elasticapm.Tracer for tracing transactions.
	// If this is nil, elasticapm.DefaultTracer will be used instead.
	Tracer *elasticapm.Tracer
}

// ServeHTTP delegates to h.Handler, tracing the transaction with
// h.Tracer, or elasticapm.DefaultTracer if h.Tracer is nil.
func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	t := h.Tracer
	if t == nil {
		t = elasticapm.DefaultTracer
	}

	var name string
	if h.RequestName != nil {
		name = h.RequestName(req)
	} else {
		name = RequestName(req)
	}
	tx := t.StartTransaction(name, "request")
	ctx := elasticapm.ContextWithTransaction(req.Context(), tx)
	req = req.WithContext(ctx)

	// TODO(axw) optimise allocations

	rw := newResponseWriter(w)
	w = wrapResponseWriter(rw)

	var finished bool
	defer func() {
		duration := time.Since(tx.Timestamp)
		if h.Recovery != nil {
			if v := recover(); v != nil {
				h.Recovery(rw, req, tx, v)
			}
		}
		tx.Result = StatusCodeString(rw.statusCode)
		if tx.Sampled() {
			tx.Context = RequestContext(req)
			tx.Context.Response = &model.Response{
				StatusCode:  rw.statusCode,
				Headers:     ResponseHeaders(rw),
				HeadersSent: &rw.written,
				Finished:    &finished,
			}
		}
		tx.Done(duration)
	}()
	h.Handler.ServeHTTP(w, req)
	finished = true
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool

	closeNotify func() <-chan bool
	flush       func()
}

func newResponseWriter(in http.ResponseWriter) *responseWriter {
	out := &responseWriter{ResponseWriter: in, statusCode: http.StatusOK}
	if in, ok := in.(http.CloseNotifier); ok {
		out.closeNotify = in.CloseNotify
	}
	if in, ok := in.(http.Flusher); ok {
		out.flush = in.Flush
	}
	return out
}

// WriteHeader sets w.statusCode and w.written, and calls through
// to the embedded ResponseWriter.
func (w *responseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
	w.written = true
}

// Write sets w.written, and calls through to the embedded ResponseWriter.
func (w *responseWriter) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	w.written = true
	return n, err
}

// CloseNotify returns w.closeNotify() if w.closeNotify is non-nil,
// otherwise it returns nil.
func (w *responseWriter) CloseNotify() <-chan bool {
	if w.closeNotify != nil {
		return w.closeNotify()
	}
	return nil
}

// Flush calls w.flush() if w.flush is non-nil, otherwise
// it does nothing.
func (w *responseWriter) Flush() {
	if w.flush != nil {
		w.flush()
	}
}

// wrapResponseWriter wraps a responseWriter so that the Pusher and Hijacker
// interfaces remain implemented by the http.ResponseWriter presented to
// the underlying http.Handler.
func wrapResponseWriter(w *responseWriter) http.ResponseWriter {
	h, _ := w.ResponseWriter.(http.Hijacker)
	p, _ := w.ResponseWriter.(http.Pusher)
	switch {
	case h != nil && p != nil:
		return responseWriterHijackerPusher{
			responseWriter: w,
			Hijacker:       h,
			Pusher:         p,
		}
	case h != nil:
		return responseWriterHijacker{
			responseWriter: w,
			Hijacker:       h,
		}
	case p != nil:
		return responseWriterPusher{
			responseWriter: w,
			Pusher:         p,
		}
	}
	return w
}

type responseWriterHijacker struct {
	*responseWriter
	http.Hijacker
}

type responseWriterPusher struct {
	*responseWriter
	http.Pusher
}

type responseWriterHijackerPusher struct {
	*responseWriter
	http.Hijacker
	http.Pusher
}
