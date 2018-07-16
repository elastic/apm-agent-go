package apmot_test

import (
	"testing"

	"github.com/elastic/apm-agent-go/module/apmot"
	"github.com/elastic/apm-agent-go/transport/transporttest"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/require"
)

func TestStartSpanRemoteParent(t *testing.T) {
	apmtracer1, recorder1 := transporttest.NewRecorderTracer()
	apmtracer2, recorder2 := transporttest.NewRecorderTracer()
	defer apmtracer1.Close()
	defer apmtracer2.Close()
	tracer1 := apmot.New(apmtracer1)
	tracer2 := apmot.New(apmtracer2)

	parentSpan := tracer1.StartSpan("parent")
	childSpan := tracer2.StartSpan("child", opentracing.ChildOf(parentSpan.Context()))
	childSpan.Finish()
	parentSpan.Finish()

	apmtracer1.Flush(nil)
	apmtracer2.Flush(nil)
	require.Len(t, recorder1.Payloads(), 1)
	require.Len(t, recorder2.Payloads(), 1)
}
