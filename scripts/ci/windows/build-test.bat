::
:: This script runs the build and test
::

mkdir build

go get -v -u github.com/jstemmer/go-junit-report
go test -v ./... > build\\test.out 2>&1

type build\\test.out
type build\\test.out | go-junit-report > build\\junit-apm-agent-go.xml
type build\\junit-apm-agent-go.xml
