.PHONY: all build test docker

all: build test docker

build:
	mkdir -p builds/
	go build -o builds/ ./cmd/...

test:
	go test -race -count=1 -v ./...

docker:
	docker build --no-cache . -t checkup