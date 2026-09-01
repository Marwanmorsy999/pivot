.PHONY: build test lint clean run help fmt vet tidy install build-all

BINARY_NAME=pivot
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"
PKG=./cmd/pivot

default: help

## build: Build the binary (requires CGO for SQLite)
build:
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BINARY_NAME) $(PKG)

## build-all: Build for multiple platforms (CGO)
build-all:
	mkdir -p dist
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 $(PKG)
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 $(PKG)
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 $(PKG)
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe $(PKG)

## test: Run all tests with race detector
test:
	CGO_ENABLED=1 go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

## test-short: Run tests without race detector
test-short:
	CGO_ENABLED=1 go test -v ./...

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
	CGO_ENABLED=1 go install $(LDFLAGS) $(PKG)

## help: Show this help
help:
	@grep -E '^## [a-zA-Z_-]+:.*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ": "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' | sed 's/^## //'
