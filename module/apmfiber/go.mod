module go.elastic.co/apm/module/apmfiber/v2

require (
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/elastic/go-licenser v0.4.1 // indirect
	github.com/elastic/go-sysinfo v1.9.0 // indirect
	github.com/gofiber/fiber/v2 v2.42.0
	github.com/jcchavezs/porto v0.4.0 // indirect
	github.com/klauspost/compress v1.15.15 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/rivo/uniseg v0.4.3 // indirect
	github.com/savsgio/gotils v0.0.0-20230208104028-c358bd845dee // indirect
	github.com/stretchr/testify v1.7.0
	github.com/tinylib/msgp v1.1.8 // indirect
	github.com/valyala/fasthttp v1.44.0
	go.elastic.co/apm/module/apmfasthttp/v2 v2.2.0
	go.elastic.co/apm/module/apmhttp/v2 v2.2.0
	go.elastic.co/apm/v2 v2.2.0
	golang.org/x/tools v0.6.0 // indirect
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp

replace go.elastic.co/apm/module/apmfasthttp/v2 => ../apmfasthttp

go 1.15
