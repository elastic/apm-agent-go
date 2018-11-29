FROM alpine:latest
WORKDIR /w3c
RUN apk add --no-cache git
RUN git clone https://github.com/w3c/trace-context.git

FROM python:3
RUN pip install aiohttp
WORKDIR /w3c/trace-context
COPY --from=0 /w3c/trace-context .
EXPOSE 7777/tcp
