module tracecontexttest/v2

require go.elastic.co/apm/module/apmhttp/v2 v2.4.1

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../../module/apmhttp

go 1.15
