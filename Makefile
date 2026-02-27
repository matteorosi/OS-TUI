.PHONY: all build test lint release clean

all: build

build:
	go build ./...

test:
	go test ./...

lint:
	golangci-lint run

release:
	goreleaser release --rm-dist

clean:
	go clean ./...
