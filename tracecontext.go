package elasticapm

import (
	"encoding/hex"
	"errors"
)

var (
	errZeroTraceID = errors.New("zero trace-id is invalid")
	errZeroSpanID  = errors.New("zero span-id is invalid")
)

const (
	traceOptionsSampledFlag = 0x00000001
)

// TraceContext holds trace context for an incoming or outgoing request.
type TraceContext struct {
	// Trace identifies the trace forest.
	Trace TraceID

	// Span identifies a span: the parent span if this context
	// corresponds to an incoming request, or the current span
	// if this is an outgoing request.
	Span SpanID

	// Options holds the trace options propagated by the parent.
	Options TraceOptions
}

// TraceID identifies a trace forest.
type TraceID [16]byte

// Validate validates the trace ID.
// This will return non-nil for a zero trace ID.
func (id TraceID) Validate() error {
	if id.isZero() {
		return errZeroTraceID
	}
	return nil
}

func (id TraceID) isZero() bool {
	return id == (TraceID{})
}

// String returns id encoded as hex.
func (id TraceID) String() string {
	return hex.EncodeToString(id[:])
}

// SpanID identifies a span within a trace.
type SpanID [8]byte

// Validate validates the span ID.
// This will return non-nil for a zero span ID.
func (id SpanID) Validate() error {
	if id.isZero() {
		return errZeroSpanID
	}
	return nil
}

func (id SpanID) isZero() bool {
	return id == SpanID{}
}

// String returns id encoded as hex.
func (id SpanID) String() string {
	return hex.EncodeToString(id[:])
}

// TraceOptions describes the options for a trace.
type TraceOptions uint8

// Sampled reports whether or not the request should be traced.
func (o TraceOptions) Sampled() bool {
	return (o & traceOptionsSampledFlag) == traceOptionsSampledFlag
}

// WithSampled changes the "sampled" flag, and returns the new options without
// modifying the original value.
func (o TraceOptions) WithSampled(sampled bool) TraceOptions {
	if sampled {
		return o | traceOptionsSampledFlag
	}
	return o & (0xFF ^ traceOptionsSampledFlag)
}
