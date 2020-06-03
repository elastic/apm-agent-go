module go.elastic.co/apm/module/apmgoredisv8

go 1.14

require (
	github.com/go-redis/redis/v8 v8.0.0-beta.2
	github.com/stretchr/testify v1.4.0
	go.elastic.co/apm v1.8.0
)

replace go.elastic.co/apm => ../..
