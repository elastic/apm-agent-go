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
	if len(payloads) != 1 {
		panic(fmt.Errorf("expected 1 payload, got %d", len(payloads)))
	}
	transactions := payloads[0].Transactions()
	if len(transactions) != 1 {
		panic(fmt.Errorf("expected 1 transaction, got %d", len(transactions)))
	}
	transaction := transactions[0]
	fmt.Printf("transaction: %s/%s\n", transaction.Name, transaction.Type)
	fmt.Println("spans:")
	for _, span := range transaction.Spans {
		fmt.Printf(" - %s/%s\n", span.Name, span.Type)
	}

	// Output:
	// transaction: Parent/unknown
	// spans:
	//  - span_0/unknown
	//  - span_1/unknown
	//  - span_2/unknown
	//  - span_3/unknown
	//  - span_4/unknown
}
