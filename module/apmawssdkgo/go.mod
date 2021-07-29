module go.elastic.co/apm/module/apmawssdkgo

go 1.15

require (
	github.com/aws/aws-sdk-go v1.38.14
	github.com/stretchr/testify v1.7.0
	go.elastic.co/apm v1.13.0
	go.elastic.co/apm/module/apmhttp v1.13.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp
