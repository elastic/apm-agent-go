FROM golang:latest
ADD . /go/src/go.elastic.co/apm
ENV GO111MODULE=on
WORKDIR /go/src/go.elastic.co/apm/internal/tracecontexttest
RUN go build -o /trace-context-service main.go

FROM gcr.io/distroless/base
COPY --from=0 /trace-context-service /
EXPOSE 5000/tcp
HEALTHCHECK CMD curl -X POST -H "Content-Type: application/json" -d "{}" http://localhost:5000
CMD /trace-context-service
