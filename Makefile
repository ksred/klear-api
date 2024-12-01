.PHONY: all build clean run test test-coverage lint help

# Go parameters
BINARY_NAME=klear-api
MAIN_PATH=cmd/server/main.go
COVERAGE_PROFILE=coverage.out

# Build parameters
BUILD_DIR=build
LDFLAGS=-ldflags "-s -w"

all: clean lint test build

help:
	@echo "Available commands:"
	@echo "  make build          - Build the application"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make run            - Run the application"
	@echo "  make test           - Run tests"
	@echo "  make test-coverage  - Run tests with coverage"
	@echo "  make lint           - Run linters"
	@echo "  make all            - Clean, lint, test, and build"
	@echo "  make help           - Show this help message"

build:
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(COVERAGE_PROFILE)
	@go clean
	@echo "Clean complete"

run: build
	@echo "Running application..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

test:
	@echo "Running tests..."
	@go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=$(COVERAGE_PROFILE) ./...
	@go tool cover -html=$(COVERAGE_PROFILE)
	@echo "Coverage report generated: $(COVERAGE_PROFILE)"

lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run ./...; \
	fi

# Development tools
.PHONY: dev-tools
dev-tools:
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "Development tools installed"

# Watch mode for development (requires fswatch)
.PHONY: watch
watch:
	@if command -v fswatch >/dev/null 2>&1; then \
		echo "Watching for changes..."; \
		fswatch -o . | xargs -n1 -I{} make run; \
	else \
		echo "fswatch not installed. Please install fswatch to use watch mode."; \
		exit 1; \
	fi 