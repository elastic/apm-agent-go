module go.elastic.co/apm/module/apmfiber/v2

require (
	github.com/gofiber/fiber/v2 v2.50.0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.8.4
	github.com/valyala/fasthttp v1.50.0
	go.elastic.co/apm/module/apmfasthttp/v2 v2.4.5
	go.elastic.co/apm/module/apmhttp/v2 v2.4.5
	go.elastic.co/apm/v2 v2.4.5
)

require (
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/elastic/go-sysinfo v1.7.1 // indirect
	github.com/elastic/go-windows v1.0.0 // indirect
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/google/uuid v1.3.1 // indirect
	github.com/joeshaw/multierror v0.0.0-20140124173710-69b34d4ec901 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/procfs v0.0.0-20190425082905-87a4384529e0 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	go.elastic.co/fastjson v1.1.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	howett.net/plist v0.0.0-20181124034731-591f970eefbb // indirect
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp

replace go.elastic.co/apm/module/apmfasthttp/v2 => ../apmfasthttp

go 1.19
