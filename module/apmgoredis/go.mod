module go.elastic.co/apm/module/apmgoredis

go 1.12

require (
	github.com/go-redis/redis v6.15.3-0.20190422135758-322709108784+incompatible
	github.com/stretchr/testify v1.2.2
	go.elastic.co/apm v1.3.0
)

replace go.elastic.co/apm => ../..
