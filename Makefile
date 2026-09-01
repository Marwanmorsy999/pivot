.PHONY: build test lint clean run help

BINARY_NAME=pivot
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

default: help

## build: Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

## build-all: Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe .

## test: Run all tests
test:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

## test-short: Run tests without race detector
test-short:
	go test -v ./...

## lint: Run linter (requires golangci-lint)
lint:
	golangci-lint run ./...

## fmt: Format Go code
fmt:
	go fmt ./...

## vet: Run go vet
vet:
	go vet ./...

## tidy: Tidy go modules
tidy:
	go mod tidy

## clean: Remove build artifacts
clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	rm -rf dist/
	rm -f coverage.out

## run: Build and run pivot
run: build
	./$(BINARY_NAME)

## install: Install to GOPATH/bin
install:
	go install $(LDFLAGS) .

## help: Show this help
help:
	@grep -E '^## [a-zA-Z_-]+:.*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ": "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' | sed 's/^## //'
