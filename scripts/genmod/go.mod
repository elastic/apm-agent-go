module genmod/v2

require (
	go.elastic.co/apm/v2 v2.4.2
	golang.org/x/mod v0.5.1
)

replace go.elastic.co/apm/v2 => ../..

go 1.15
