module go.elastic.co/apm/module/apmawssdkgo/v2

go 1.15

require (
	github.com/aws/aws-sdk-go v1.38.14
	github.com/stretchr/testify v1.7.0
	go.elastic.co/apm/module/apmhttp/v2 v2.3.0
	go.elastic.co/apm/v2 v2.3.0
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp
