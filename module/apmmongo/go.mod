module go.elastic.co/apm/module/apmmongo/v2

require (
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/v2 v2.1.0
	go.mongodb.org/mongo-driver v1.5.1
)

replace go.elastic.co/apm/v2 => ../..

go 1.15
