.PHONY: all build test clean run ui-dev ui-build

BINARY_NAME=socketlens
VERSION=1.0.0

all: test build

build: ui-build
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/$(BINARY_NAME) ./cmd/socketlens

build-go:
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/$(BINARY_NAME) ./cmd/socketlens

test:
	go test -v -race ./pkg/...

test-cover:
	go test -v -coverprofile=coverage.txt -covermode=atomic ./pkg/...
	go tool cover -html=coverage.txt -o coverage.html

ui-dev:
	cd ui && npm run dev

ui-build:
	cd ui && npm install && npm run build

clean:
	rm -rf bin/ dist/ ui/dist/ coverage.txt coverage.html
