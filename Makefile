.PHONY: all build build-sql test docker

all: build test

DOCKER_IMAGE := checkup

build:
	go fmt ./...
	mkdir -p builds/
	go build -o builds/ ./cmd/...

build-sql:
	go fmt ./...
	go build -o builds/ -tags sql ./cmd/...

test:
	go test -race -count=1 ./...

docker:
	docker build --no-cache . -t $(DOCKER_IMAGE)
