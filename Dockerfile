FROM golang:1.13-alpine as builder

ENV CGO_ENABLED=0

COPY . /app
WORKDIR /app
RUN apk --no-cache add make && make build

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/builds/checkup /usr/local/bin/checkup

ENTRYPOINT ["checkup"]
