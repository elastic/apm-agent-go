module go.elastic.co/apm/module/apmmongo/v2

require (
	github.com/elastic/go-licenser v0.4.0 // indirect
	github.com/elastic/go-sysinfo v1.7.1 // indirect
	github.com/jcchavezs/porto v0.4.0 // indirect
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/v2 v2.0.0
	go.mongodb.org/mongo-driver v1.5.1
)

replace go.elastic.co/apm/v2 => ../..

go 1.15

exclude (
	gopkg.in/yaml.v2 v2.2.1
	gopkg.in/yaml.v2 v2.2.2
	gopkg.in/yaml.v2 v2.2.4
	gopkg.in/yaml.v2 v2.2.7
)
