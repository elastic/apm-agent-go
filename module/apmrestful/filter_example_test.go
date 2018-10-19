package apmrestful_test

import (
	"github.com/emicklei/go-restful"

	"go.elastic.co/apm/module/apmrestful"
)

func ExampleFilter() {
	// Install the filter into the default/global Container.
	restful.Filter(apmrestful.Filter())
}
