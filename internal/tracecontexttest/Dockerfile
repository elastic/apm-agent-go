FROM cgr.dev/chainguard/go:latest
ADD . /go/src/go.elastic.co/apm
WORKDIR /go/src/go.elastic.co/apm/internal/tracecontexttest
RUN CGO_ENABLED=0 go build -o /trace-context-service main.go

EXPOSE 5000/tcp
HEALTHCHECK CMD curl -X POST -H "Content-Type: application/json" -d "{}" http://localhost:5000
CMD [ "/trace-context-service" ]
