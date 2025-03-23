.PHONY: build install clean test

BINARY_NAME=mcp-go-debugger
VERSION=$(shell git describe --tags --always --dirty)
BUILD_DIR=bin
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/mcp-go-debugger

clean:
	rm -f $(BUILD_DIR)/$(BINARY_NAME)
	go clean

test:
	go test -v ./...

install:
	go install $(LDFLAGS) ./cmd/mcp-go-debugger

run: build
	$(BUILD_DIR)/$(BINARY_NAME)

# Create directories if they don't exist
$(shell mkdir -p $(BUILD_DIR)) 