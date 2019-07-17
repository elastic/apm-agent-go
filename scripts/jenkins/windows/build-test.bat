::
:: This script runs the build and test
::

go get -v -u github.com/jstemmer/go-junit-report
go get -t ./...
mkdir build
go test -v ./... 2>&1 | go-junit-report > build\\junit-apm-agent-go.xml
type build\\junit-apm-agent-go.xml
