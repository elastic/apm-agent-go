module tracecontexttest

require (
	go.elastic.co/apm v1.1.1
	go.elastic.co/apm/module/apmhttp v1.1.1
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../../module/apmhttp
