package apm_test

import (
	"context"
	"html/template"
	"os"

	"go.elastic.co/apm"
)

func ExampleTransaction_EnsureParent() {
	tx := apm.DefaultTracer.StartTransactionOptions("name", "type", apm.TransactionOptions{
		TraceContext: apm.TraceContext{
			Trace: apm.TraceID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
			Span:  apm.SpanID{0, 1, 2, 3, 4, 5, 6, 7},
		},
	})
	defer tx.Discard()

	tpl := template.Must(template.New("").Parse(`
<script src="elastic-apm-js-base/dist/bundles/elastic-apm-js-base.umd.min.js"></script>
<script>
  elasticApm.init({
    serviceName: '',
    serverUrl: 'http://localhost:8200',
    pageLoadTraceId: {{.TraceContext.Trace}},
    pageLoadSpanId: {{.EnsureParent}},
    pageLoadSampled: {{.Sampled}},
  })
</script>
`))

	if err := tpl.Execute(os.Stdout, tx); err != nil {
		panic(err)
	}

	// Output:
	// <script src="elastic-apm-js-base/dist/bundles/elastic-apm-js-base.umd.min.js"></script>
	// <script>
	//   elasticApm.init({
	//     serviceName: '',
	//     serverUrl: 'http://localhost:8200',
	//     pageLoadTraceId: "000102030405060708090a0b0c0d0e0f",
	//     pageLoadSpanId: "0001020304050607",
	//     pageLoadSampled:  false ,
	//   })
	// </script>
}

func ExampleTransaction_EnsureParent_nilTransaction() {
	tpl := template.Must(template.New("").Parse(`
<script>
elasticApm.init({
  {{.TraceContext.Trace}},
  {{.EnsureParent}},
  {{.Sampled}},
})
</script>
`))

	// Demonstrate that Transaction's TraceContext, EnsureParent,
	// and Sampled methods will not panic when called with a nil
	// Transaction.
	tx := apm.TransactionFromContext(context.Background())
	if err := tpl.Execute(os.Stdout, tx); err != nil {
		panic(err)
	}

	// Output:
	// <script>
	// elasticApm.init({
	//   "00000000000000000000000000000000",
	//   "0000000000000000",
	//    false ,
	// })
	// </script>
}
