module genmod

require (
	github.com/pkg/errors v0.9.1
	go.elastic.co/apm/v2 v2.1.0
	golang.org/x/mod v0.5.1
)

replace go.elastic.co/apm/v2 => ../..

go 1.15
