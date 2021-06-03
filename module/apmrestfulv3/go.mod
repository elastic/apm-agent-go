module go.elastic.co/apm/module/apmrestfulv3

require (
	github.com/emicklei/go-restful v2.9.6+incompatible
	github.com/emicklei/go-restful/v3 v3.5.1
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm v1.12.0
	go.elastic.co/apm/module/apmhttp v1.12.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp

go 1.13
