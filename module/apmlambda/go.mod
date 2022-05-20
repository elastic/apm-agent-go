module go.elastic.co/apm/module/apmlambda/v2

require (
	github.com/aws/aws-lambda-go v1.8.0
	go.elastic.co/apm/v2 v2.1.0
)

replace go.elastic.co/apm/v2 => ../..

go 1.15
