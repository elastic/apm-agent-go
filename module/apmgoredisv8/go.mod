module go.elastic.co/apm/module/apmgoredisv8/v2

go 1.14

require (
	github.com/go-redis/redis/v8 v8.0.0-beta.2
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/v2 v2.0.0
)

replace go.elastic.co/apm/v2 => ../..
