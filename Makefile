.PHONY: all build test

all: build test

build:
	mkdir -p builds/
	go build -o builds/ ./cmd/...

test:
	go test -race -count=1 -v ./...
