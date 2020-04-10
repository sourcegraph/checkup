.PHONY: all build test docker

all: build test

DOCKER_IMAGE := checkup

build:
	mkdir -p builds/
	go build -o builds/ ./cmd/...

test:
	/root/go/bin/gotest -race -count=1 -v ./...

docker:
	docker build --no-cache . -t $(DOCKER_IMAGE)
