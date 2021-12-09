module tracecontexttest

require go.elastic.co/apm/module/apmhttp/v2 v2.0.0

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../../module/apmhttp

go 1.13
