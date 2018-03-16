// +build ignore

package main

import (
	"fmt"
	"time"

	"github.com/elastic/apm-agent-go"
	_ "github.com/elastic/apm-agent-go/contrib/apmot"
	opentracing "github.com/opentracing/opentracing-go"
)

func main() {
	parent := opentracing.StartSpan("Parent")
	for i := 0; i < 20; i++ {
		parent.LogEvent(fmt.Sprintf("Starting child #%d", i))
		child := opentracing.StartSpan("Child", opentracing.ChildOf(parent.Context()))
		time.Sleep(50 * time.Millisecond)
		child.Finish()
	}
	parent.LogEvent("A Log")
	parent.Finish()
	elasticapm.DefaultTracer.Flush(nil)
}
