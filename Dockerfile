FROM golang:latest

COPY . /project

WORKDIR /project/cmd/checkup

RUN go get -v -d;

RUN go build -v -ldflags '-s' -o ../../checkup

WORKDIR /project

ENTRYPOINT ["./checkup"]
