module go.elastic.co/apm/module/apmechov4

require (
	github.com/labstack/echo/v4 v4.0.0
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.3.0
	go.elastic.co/apm v1.5.0
	go.elastic.co/apm/module/apmhttp v1.5.0
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4 // indirect
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp

go 1.13
