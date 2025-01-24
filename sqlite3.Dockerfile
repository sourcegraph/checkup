#---------------------------- Build 
FROM golang:1.14-alpine as builder

COPY . /app
WORKDIR /app
RUN apk add --no-cache make gcc musl-dev && \
    rm -rf /var/cache/apk/* && \
    make build-sqlite3

#---------------------------- Deploy 
FROM alpine:3.4

RUN apk --update upgrade && \
    apk add --no-cache sqlite ca-certificates && \
    rm -f /usr/bin/sqlite3 && \
    rm -rf /var/cache/apk/*

RUN mkdir /lib64 && \
    ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

WORKDIR /app

COPY --from=builder /app/builds/checkup /usr/local/bin/checkup
ADD statuspage/ /app/statuspage

RUN addgroup -g 1000 app 
RUN adduser -g "" -G app -D -H -u 1000 app 

USER app

EXPOSE 3000
ENTRYPOINT ["checkup"]
CMD ["serve"]