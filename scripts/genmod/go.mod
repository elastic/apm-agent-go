module genmod

require (
	github.com/pkg/errors v0.8.1
	go.elastic.co/apm v1.7.1
)

replace go.elastic.co/apm => ../..

go 1.13
