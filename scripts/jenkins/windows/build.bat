::
:: This script runs the build
::

go get -v -u github.com/jstemmer/go-junit-report
go get -t ./...
mkdir build
go test -v ./... 2>&1 | go-junit-report > build\\junit-apm-agent-go.xml
