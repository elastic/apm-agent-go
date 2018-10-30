FROM golang:latest
WORKDIR /go/src/go.elastic.co/apm
RUN go get -v github.com/pkg/errors
RUN go get -v go.elastic.co/fastjson
RUN go get -v golang.org/x/net/context
RUN go get -v golang.org/x/net/context/ctxhttp

ADD . /go/src/go.elastic.co/apm
EXPOSE 5000/tcp
CMD go run internal/tracecontexttest/main.go
