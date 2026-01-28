.PHONY: build build-all run clean install test

# Binary name
BINARY=portsweep

# Version (can be overridden: make build VERSION=1.0.0)
VERSION ?= dev
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

# Build for current platform
build:
	go build $(LDFLAGS) -o $(BINARY) .

# Run tests
test:
	go test -v ./...

# Run the application
run:
	go run .

# Build for macOS (both architectures)
build-macos: build-macos-arm64 build-macos-amd64

build-macos-arm64:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY)-darwin-arm64 .

build-macos-amd64:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-darwin-amd64 .

# Build universal binary for macOS
build-macos-universal: build-macos-arm64 build-macos-amd64
	lipo -create -output $(BINARY)-darwin-universal $(BINARY)-darwin-arm64 $(BINARY)-darwin-amd64

# Build for all platforms (Linux builds included but not officially supported in v0.1.0)
build-all: build-macos
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY)-linux-arm64 .

# Install to /usr/local/bin
install: build
	cp $(BINARY) /usr/local/bin/$(BINARY)

# Clean build artifacts
clean:
	rm -f $(BINARY) $(BINARY)-*
