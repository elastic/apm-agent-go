module go.elastic.co/apm/module/apmgoredis

go 1.12

require (
	github.com/go-redis/redis v6.15.3-0.20190424063336-97e6ed817821+incompatible
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm v1.14.0
	gopkg.in/yaml.v2 v2.2.2 // indirect
)

replace go.elastic.co/apm => ../..
