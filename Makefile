# Makefile for encoder project

# Binary name
BINARY_NAME=encoder
OUTPUT_DIR=.

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-s -w"

.PHONY: all build clean test coverage cover cover-html run help tidy deps fmt vet check bench stats install

# Default target
all: clean build

# Build the project
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(OUTPUT_DIR)/$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -f $(OUTPUT_DIR)/$(BINARY_NAME)
	@rm -f $(OUTPUT_DIR)/encoder_bin
	@rm -f $(OUTPUT_DIR)/coverage.out
	@rm -f $(OUTPUT_DIR)/coverage.html
	@rm -f /tmp/encoded_*.opus
	@rm -f /tmp/test_audio_*.opus
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(OUTPUT_DIR)/$(BINARY_NAME)

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy
	@echo "Dependencies tidied"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	@echo "Dependencies installed"

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...
	@echo "Code formatted"

# Vet code
vet:
	@echo "Vetting code..."
	$(GOCMD) vet ./...
	@echo "Code vetted"

# Full check (format, vet, test)
check: fmt vet test
	@echo "All checks passed"

# Quick coverage report (terminal output only)
cover:
	@echo "Running tests with coverage..."
	@$(GOTEST) -coverprofile=coverage.out ./... > /dev/null 2>&1
	@$(GOCMD) tool cover -func=coverage.out

# View coverage in browser
cover-html: coverage
	@echo "Opening coverage report in browser..."
	@which xdg-open > /dev/null && xdg-open coverage.html || \
	 which open > /dev/null && open coverage.html || \
	 echo "Please open coverage.html manually"

# Benchmark tests
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# Show project statistics
stats:
	@echo "Project Statistics:"
	@echo "===================="
	@echo "Go files:       $$(find . -name '*.go' -not -path './vendor/*' | wc -l)"
	@echo "Test files:     $$(find . -name '*_test.go' | wc -l)"
	@echo "Total lines:    $$(find . -name '*.go' -not -path './vendor/*' | xargs wc -l | tail -1 | awk '{print $$1}')"
	@echo "Test lines:     $$(find . -name '*_test.go' | xargs wc -l | tail -1 | awk '{print $$1}')"
	@echo ""
	@echo "Package breakdown:"
	@for pkg in $$(go list ./...); do \
		echo "  $$pkg"; \
	done

# Install project binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin..."
	@cp $(OUTPUT_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "Installed successfully"

# Help target
help:
	@echo "Available targets:"
	@echo ""
	@echo "Building:"
	@echo "  make build      - Build the project"
	@echo "  make install    - Build and install to GOPATH/bin"
	@echo "  make run        - Build and run the application"
	@echo ""
	@echo "Testing:"
	@echo "  make test       - Run all tests"
	@echo "  make coverage   - Run tests with HTML coverage report"
	@echo "  make cover      - Run tests with terminal coverage summary"
	@echo "  make cover-html - Generate and open HTML coverage report"
	@echo "  make bench      - Run benchmark tests"
	@echo ""
	@echo "Code Quality:"
	@echo "  make fmt        - Format code"
	@echo "  make vet        - Run go vet"
	@echo "  make check      - Run fmt, vet, and test"
	@echo ""
	@echo "Dependencies:"
	@echo "  make deps       - Download dependencies"
	@echo "  make tidy       - Tidy go.mod and go.sum"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean      - Remove build artifacts and generated files"
	@echo "  make stats      - Show project statistics"
	@echo "  make all        - Clean and build (default)"
	@echo "  make help       - Show this help message"
