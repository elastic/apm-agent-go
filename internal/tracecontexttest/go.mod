module tracecontexttest

require go.elastic.co/apm/module/apmhttp v1.7.2

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../../module/apmhttp

go 1.13
