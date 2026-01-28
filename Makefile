.PHONY: build build-all run clean install

# Binary name
BINARY=portsweep

# Build for current platform
build:
	go build -o $(BINARY) .

# Run the application
run:
	go run .

# Build for macOS (both architectures)
build-macos: build-macos-arm64 build-macos-amd64

build-macos-arm64:
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY)-darwin-arm64 .

build-macos-amd64:
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY)-darwin-amd64 .

# Build universal binary for macOS
build-macos-universal: build-macos-arm64 build-macos-amd64
	lipo -create -output $(BINARY)-darwin-universal $(BINARY)-darwin-arm64 $(BINARY)-darwin-amd64

# Build for all platforms
build-all: build-macos
	GOOS=linux GOARCH=amd64 go build -o $(BINARY)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -o $(BINARY)-linux-arm64 .

# Install to /usr/local/bin
install: build
	cp $(BINARY) /usr/local/bin/$(BINARY)

# Clean build artifacts
clean:
	rm -f $(BINARY) $(BINARY)-*
