module go.elastic.co/apm/module/apmgoredisv8/v2

go 1.15

require (
	github.com/go-redis/redis/v8 v8.11.4
	github.com/stretchr/testify v1.7.0
	go.elastic.co/apm/v2 v2.0.0
)

replace go.elastic.co/apm/v2 => ../..
