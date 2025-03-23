.PHONY: build clean test install

BINARY_NAME=go-debugger-mcp
VERSION=0.1.0
BUILD_DIR=./bin
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/go-debugger-mcp

clean:
	rm -rf $(BUILD_DIR)
	go clean

test:
	go test -v ./...

install: build
	go install $(LDFLAGS) ./cmd/go-debugger-mcp

run: build
	$(BUILD_DIR)/$(BINARY_NAME)

# Create directories if they don't exist
$(shell mkdir -p $(BUILD_DIR)) 