package apmtest

import (
	"log"

	"go.elastic.co/apm"
	"go.elastic.co/apm/transport/transporttest"
)

// DiscardTracer is an apm.Tracer that discards all events.
//
// This tracer may be used by multiple tests, and so should
// not be modified or closed.
var DiscardTracer *apm.Tracer

func init() {
	tracer, err := apm.NewTracer("", "")
	if err != nil {
		log.Fatal(err)
	}
	tracer.Transport = transporttest.Discard
	DiscardTracer = tracer
}
