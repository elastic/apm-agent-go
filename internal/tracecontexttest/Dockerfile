FROM golang:latest
ADD . /go/src/go.elastic.co/apm
ENV GO111MODULE=on
ENV GOPROXY=https://proxy.golang.org
WORKDIR /go/src/go.elastic.co/apm/internal/tracecontexttest
RUN go build -o /trace-context-service main.go

EXPOSE 5000/tcp
HEALTHCHECK CMD curl -X POST -H "Content-Type: application/json" -d "{}" http://localhost:5000
CMD /trace-context-service
