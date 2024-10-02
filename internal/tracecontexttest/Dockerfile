FROM cgr.dev/chainguard/go:latest AS builder
ADD . /go/src/go.elastic.co/apm
WORKDIR /go/src/go.elastic.co/apm/internal/tracecontexttest
RUN CGO_ENABLED=0 go build -o /trace-context-service main.go

FROM cgr.dev/chainguard/curl:latest

COPY --chmod=0644 --chown=nonroot:nonroot --from=builder /trace-context-service /trace-context-service

EXPOSE 5000/tcp
HEALTHCHECK CMD curl -X POST -H "Content-Type: application/json" -d "{}" http://localhost:5000
CMD [ "/trace-context-service" ]
