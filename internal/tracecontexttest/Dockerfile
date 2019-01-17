FROM golang:latest
ADD . /go/src/go.elastic.co/apm
ENV GO111MODULE=on

EXPOSE 5000/tcp
WORKDIR /go/src/go.elastic.co/apm/internal/tracecontexttest
HEALTHCHECK CMD curl -X POST -H "Content-Type: application/json" -d "{}" http://localhost:5000
CMD go run main.go
