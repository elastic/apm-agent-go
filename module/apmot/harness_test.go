package apmot

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/harness"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/module/apmhttp"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

func TestHarness(t *testing.T) {
	// NOTE(axw) we do not support binary propagation, but we patch in
	// basic support *for the tests only* so we can check compatibility
	// with the HTTP and text formats.
	binaryInject = func(w io.Writer, traceContext elasticapm.TraceContext) error {
		return json.NewEncoder(w).Encode(apmhttp.FormatTraceparentHeader(traceContext))
	}
	binaryExtract = func(r io.Reader) (elasticapm.TraceContext, error) {
		var headerValue string
		if err := json.NewDecoder(r).Decode(&headerValue); err != nil {
			return elasticapm.TraceContext{}, err
		}
		return apmhttp.ParseTraceparentHeader(headerValue)
	}
	defer func() {
		binaryInject = binaryInjectUnsupported
		binaryExtract = binaryExtractUnsupported
	}()

	newTracer := func() (opentracing.Tracer, func()) {
		apmtracer, err := elasticapm.NewTracer("transporttest", "")
		if err != nil {
			panic(err)
		}
		apmtracer.Transport = transporttest.Discard
		tracer := New(WithTracer(apmtracer))
		return tracer, apmtracer.Close
	}
	harness.RunAPIChecks(t, newTracer,
		harness.CheckExtract(true),
		harness.CheckInject(true),
		harness.UseProbe(harnessAPIProbe{}),
	)
}

type harnessAPIProbe struct{}

func (harnessAPIProbe) SameTrace(first, second opentracing.Span) bool {
	ctx1, ok := first.Context().(spanContext)
	if !ok {
		return false
	}
	ctx2, ok := second.Context().(spanContext)
	if !ok {
		return false
	}
	return ctx1.traceContext.Trace == ctx2.traceContext.Trace
}

func (harnessAPIProbe) SameSpanContext(span opentracing.Span, sc opentracing.SpanContext) bool {
	ctx1, ok := span.Context().(spanContext)
	if !ok {
		return false
	}
	ctx2, ok := sc.(spanContext)
	if !ok {
		return false
	}
	return ctx1.traceContext == ctx2.traceContext
}
