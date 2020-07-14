.PHONY: all build fmt test docker

all: build test

DOCKER_IMAGE := checkup

build: fmt
	go build -o builds/ ./cmd/...

build-%: TAG=$*
build-%: fmt
	go build -o builds/ -tags $(TAG) ./cmd/...

fmt:
	mkdir -p builds/
	go fmt ./...
	go mod tidy

test:
	go test -race -count=1 ./...

test-%: TAG=$*
test-%:
	go test -tags $(TAG) -race -count=1 ./...

docker:
	docker build --no-cache . -t $(DOCKER_IMAGE)
