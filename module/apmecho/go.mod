module go.elastic.co/apm/module/apmecho

require (
	github.com/labstack/echo v3.3.5+incompatible
	github.com/labstack/gommon v0.2.8 // indirect
	github.com/mattn/go-colorable v0.0.9 // indirect
	github.com/mattn/go-isatty v0.0.4 // indirect
	github.com/pkg/errors v0.8.0
	github.com/stretchr/testify v1.2.2
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v0.0.0-20170224212429-dcecefd839c4 // indirect
	go.elastic.co/apm v1.1.1
	go.elastic.co/apm/module/apmhttp v1.1.1
	golang.org/x/crypto v0.0.0-20181203042331-505ab145d0a9 // indirect
)

replace go.elastic.co/apm => ../../

replace go.elastic.co/apm/module/apmhttp => ../apmhttp
