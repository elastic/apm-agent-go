package apmot_test

import (
	"fmt"
	"time"

	"github.com/opentracing/opentracing-go"
)

func Example() {
	tracer, apmtracer, recorder := newTestTracer()
	defer apmtracer.Close()
	opentracing.SetGlobalTracer(tracer)
	defer opentracing.SetGlobalTracer(nil)

	parent := opentracing.StartSpan("Parent")
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("span_%d", i)
		parent.LogEvent(fmt.Sprintf("Starting %s", id))
		child := opentracing.StartSpan(id, opentracing.ChildOf(parent.Context()))
		time.Sleep(10 * time.Millisecond)
		child.Finish()
	}
	parent.LogEvent("A Log")
	parent.Finish()
	apmtracer.Flush(nil)

	payloads := recorder.Payloads()
	if len(payloads.Transactions) != 1 {
		panic(fmt.Errorf("expected 1 transaction, got %d", len(payloads.Transactions)))
	}
	for _, transaction := range payloads.Transactions {
		fmt.Printf("transaction: %s/%s\n", transaction.Name, transaction.Type)
	}
	for _, span := range payloads.Spans {
		fmt.Printf("span: %s/%s\n", span.Name, span.Type)
	}

	// Output:
	// transaction: Parent/custom
	// span: span_0/custom
	// span: span_1/custom
	// span: span_2/custom
	// span: span_3/custom
	// span: span_4/custom
}
