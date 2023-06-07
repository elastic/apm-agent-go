module tracecontexttest/v2

require go.elastic.co/apm/module/apmhttp/v2 v2.4.2

require (
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/elastic/go-sysinfo v1.7.1 // indirect
	github.com/elastic/go-windows v1.0.0 // indirect
	github.com/joeshaw/multierror v0.0.0-20140124173710-69b34d4ec901 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/procfs v0.0.0-20190425082905-87a4384529e0 // indirect
	go.elastic.co/apm/v2 v2.4.2 // indirect
	go.elastic.co/fastjson v1.1.0 // indirect
	golang.org/x/sys v0.0.0-20220209214540-3681064d5158 // indirect
	howett.net/plist v0.0.0-20181124034731-591f970eefbb // indirect
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../../module/apmhttp

go 1.19
