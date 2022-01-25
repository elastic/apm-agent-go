module go.elastic.co/apm/module/apmgoredis/v2

go 1.12

require (
	github.com/elastic/go-licenser v0.4.0 // indirect
	github.com/elastic/go-sysinfo v1.7.1 // indirect
	github.com/go-redis/redis v6.15.3-0.20190424063336-97e6ed817821+incompatible
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/jcchavezs/porto v0.4.0 // indirect
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/v2 v2.0.0
	golang.org/x/tools v0.1.8 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace go.elastic.co/apm/v2 => ../..

exclude (
	gopkg.in/yaml.v2 v2.2.1
	gopkg.in/yaml.v2 v2.2.2
	gopkg.in/yaml.v2 v2.2.4
	gopkg.in/yaml.v2 v2.2.7
)
