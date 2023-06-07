module go.elastic.co/apm/module/apmrestfulv3/v2

require (
	github.com/emicklei/go-restful/v3 v3.8.0
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/module/apmhttp/v2 v2.4.2
	go.elastic.co/apm/v2 v2.4.2
)

require (
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/elastic/go-sysinfo v1.7.1 // indirect
	github.com/elastic/go-windows v1.0.0 // indirect
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/joeshaw/multierror v0.0.0-20140124173710-69b34d4ec901 // indirect
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v0.0.0-20180701023420-4b7aa43c6742 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/procfs v0.0.0-20190425082905-87a4384529e0 // indirect
	go.elastic.co/fastjson v1.1.0 // indirect
	golang.org/x/sys v0.0.0-20220209214540-3681064d5158 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
	howett.net/plist v0.0.0-20181124034731-591f970eefbb // indirect
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp

go 1.19
