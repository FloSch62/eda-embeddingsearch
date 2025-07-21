.PHONY: all build clean test install lint fmt vet

# Binary name
BINARY_NAME=embeddingsearch

# Build variables
GO=go
GOFLAGS=-ldflags "-s -w"
GOBUILD=$(GO) build $(GOFLAGS)
GOCLEAN=$(GO) clean
GOTEST=$(GO) test
GOGET=$(GO) get
GOFMT=$(GO) fmt
GOVET=$(GO) vet

# Directories
CMD_DIR=./cmd/embeddingsearch
BIN_DIR=./bin

# Platform-specific variables
LINUX_DIR=$(BIN_DIR)/linux
DARWIN_DIR=$(BIN_DIR)/darwin
WINDOWS_DIR=$(BIN_DIR)/win32

all: clean build

build: build-linux build-darwin build-windows

build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(LINUX_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(LINUX_DIR)/$(BINARY_NAME) $(CMD_DIR)

build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(DARWIN_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(DARWIN_DIR)/$(BINARY_NAME) $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(DARWIN_DIR)/$(BINARY_NAME)-arm64 $(CMD_DIR)

build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(WINDOWS_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(WINDOWS_DIR)/$(BINARY_NAME).exe $(CMD_DIR)

# Build only for current platform
build-local:
	@echo "Building for current platform..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)

clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BIN_DIR)

test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.txt -covermode=atomic ./...

test-short:
	@echo "Running short tests..."
	$(GOTEST) -v -short ./...

coverage: test
	@echo "Generating coverage report..."
	$(GO) tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

install: build-local
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BIN_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME) || cp $(BIN_DIR)/$(BINARY_NAME) ~/go/bin/$(BINARY_NAME)
	@echo "$(BINARY_NAME) installed to GOPATH/bin"

lint:
	@echo "Running linters..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run

fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Check for common issues
check: fmt vet
	@echo "Running basic checks..."
	@echo "All checks passed!"

# Development helpers
dev-deps:
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

run:
	@echo "Running with sample query..."
	$(GO) run $(CMD_DIR) "show interfaces"

# Help
help:
	@echo "Available targets:"
	@echo "  make build        - Build for all platforms"
	@echo "  make build-local  - Build for current platform only"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make test         - Run tests with coverage"
	@echo "  make test-short   - Run short tests"
	@echo "  make coverage     - Generate coverage report"
	@echo "  make bench        - Run benchmarks"
	@echo "  make install      - Install binary to GOPATH/bin"
	@echo "  make lint         - Run linters"
	@echo "  make fmt          - Format code"
	@echo "  make vet          - Run go vet"
	@echo "  make check        - Run basic checks (fmt, vet)"
	@echo "  make dev-deps     - Install development dependencies"
	@echo "  make run          - Run with sample query"
	@echo "  make help         - Show this help message"