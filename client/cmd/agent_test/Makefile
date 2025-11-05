.PHONY: build run clean help

# Binary name
BINARY=agent_test

# Build parameters
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY)..."
	@go build -o $(BINARY) .
	@echo "Build complete: ./$(BINARY)"

# Build and run with default settings
run: build
	@./$(BINARY)

# Run with custom parameters
run-local: build
	@./$(BINARY) -url http://localhost:8080 -kb $(KB_ID)

# Run with existing session
run-session: build
	@./$(BINARY) -url http://localhost:8080 -session $(SESSION_ID)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY)
	@echo "Clean complete"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies installed"

# Run tests
test:
	@go test -v ./...

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@GOOS=linux GOARCH=amd64 go build -o $(BINARY)-linux-amd64 .
	@GOOS=darwin GOARCH=amd64 go build -o $(BINARY)-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 go build -o $(BINARY)-darwin-arm64 .
	@GOOS=windows GOARCH=amd64 go build -o $(BINARY)-windows-amd64.exe .
	@echo "Build complete for all platforms"

# Help
help:
	@echo "Available targets:"
	@echo "  make build          - Build the binary"
	@echo "  make run            - Build and run with no parameters"
	@echo "  make run-local      - Build and run with local server (requires KB_ID env var)"
	@echo "  make run-session    - Build and run with existing session (requires SESSION_ID env var)"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make deps           - Install dependencies"
	@echo "  make test           - Run tests"
	@echo "  make build-all      - Build for multiple platforms"
	@echo "  make help           - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  KB_ID=kb_123 make run-local"
	@echo "  SESSION_ID=sess_456 make run-session"
	@echo "  ./$(BINARY) -url http://localhost:8080 -kb kb_123 -token mytoken"

