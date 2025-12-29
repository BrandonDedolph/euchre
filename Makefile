.PHONY: build test run clean lint help

# Default target
all: build

# Build the binary
build:
	go build -o bin/euchre ./cmd/euchre

# Run the application
run:
	go run ./cmd/euchre

# Run tests
test:
	go test -v ./...

# Run tests with race detector
test-race:
	go test -race ./...

# Generate test coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Install dependencies
deps:
	go mod download
	go mod tidy

# Show help
help:
	@echo "Euchre - Learn and play the classic card game"
	@echo ""
	@echo "Targets:"
	@echo "  build     - Build the binary to bin/euchre"
	@echo "  run       - Run the application"
	@echo "  test      - Run tests"
	@echo "  test-race - Run tests with race detector"
	@echo "  coverage  - Generate test coverage report"
	@echo "  lint      - Run linter"
	@echo "  clean     - Remove build artifacts"
	@echo "  deps      - Download and tidy dependencies"
	@echo "  help      - Show this help"
