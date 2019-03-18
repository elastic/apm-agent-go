module go.elastic.co/apm/module/apmecho

require (
	github.com/labstack/echo v3.3.10+incompatible
	github.com/labstack/gommon v0.2.8 // indirect
	github.com/mattn/go-colorable v0.1.1 // indirect
	github.com/mattn/go-isatty v0.0.7 // indirect
	github.com/pkg/errors v0.8.0
	github.com/stretchr/testify v1.3.0
	github.com/valyala/fasttemplate v1.0.1 // indirect
	go.elastic.co/apm v1.2.0
	go.elastic.co/apm/module/apmhttp v1.2.0
	golang.org/x/crypto v0.0.0-20190313024323-a1f597ede03a // indirect
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp
