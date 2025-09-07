# Makefile for SQL Graph Visualizer
# Requires Go 1.24+

.PHONY: help install generate format test build run clean docker-up docker-down sec-scan ci-check dev quick lint

# Default target
help:
	@echo "SQL Graph Visualizer - Development Commands"
	@echo ""
	@echo "Available targets:"
	@echo "  install    - Install dependencies and tools"
	@echo "  generate   - Generate GraphQL code"
	@echo "  format     - Format Go code"
	@echo "  lint       - Run golangci-lint"
	@echo "  test       - Run tests"
	@echo "  build      - Build the application"
	@echo "  run        - Run the application"
	@echo "  clean      - Clean build artifacts"
	@echo "  docker-up  - Start Docker services (Neo4j)"
	@echo "  docker-down- Stop Docker services"
	@echo "  sec-scan   - Run security scans (govulncheck, gosec)"
	@echo "  ci-check   - Run CI checks locally"
	@echo ""

# Install dependencies and tools
install:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy
	@echo "Installing gqlgen..."
	go install github.com/99designs/gqlgen@v0.17.78
	@echo "Installing golangci-lint..."
	@if [ ! -f $(HOME)/go/bin/golangci-lint ]; then \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(HOME)/go/bin v1.55.2; \
	fi
	@echo "Installation complete"

# Generate GraphQL code
generate:
	@echo "Generating GraphQL code..."
	$(HOME)/go/bin/gqlgen generate
	@echo "GraphQL code generated"

# Format Go code
format:
	@echo "Formatting Go code..."
	gofmt -s -w .
	@echo "Code formatted"

# Run tests
test:
	@echo "Running unit tests..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic \
		$(shell go list ./... | grep -v '/internal/tests/integration')
	@echo "Tests completed"

# Run integration tests (requires Docker services)
test-integration:
	@echo "Running integration tests..."
	go test -v -timeout 15m -tags=integration \
		-coverprofile=integration-coverage.out \
		-covermode=atomic \
		./internal/tests/integration/...
	@echo "Integration tests completed"

# Build the application
build:
	@echo "Building application..."
	go build -o sql-graph-visualizer cmd/main.go
	@echo "Build completed"

# Run the application
run:
	@echo "Starting application..."
	go run cmd/main.go

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f sql-graph-visualizer
	rm -f coverage.out
	rm -f integration-coverage.out
	@echo "Clean completed"

# Start Docker services
docker-up:
	@echo "Starting Docker services..."
	docker-compose up -d neo4j-test
	@echo "Docker services started"

# Stop Docker services
docker-down:
	@echo "Stopping Docker services..."
	docker-compose down
	@echo "Docker services stopped"

# Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	$(HOME)/go/bin/golangci-lint run --timeout=10m
	@echo "Lint completed"

# Run CI checks locally
ci-check: install generate format
	@echo "Running CI checks locally..."
	@echo "Checking Go modules consistency..."
	go mod tidy
	@if [ -n "$$(git diff go.mod go.sum)" ]; then \
		echo "go.mod or go.sum are not up to date"; \
		git diff go.mod go.sum; \
		exit 1; \
	fi
	@echo "Checking code formatting..."
	@if [ -n "$$(gofmt -s -l .)" ]; then \
		echo "Code is not formatted properly:"; \
		gofmt -s -d .; \
		exit 1; \
	fi
	@echo "Running go vet..."
	go vet ./...
	@echo "Running golangci-lint..."
	$(HOME)/go/bin/golangci-lint run --timeout=10m
	@echo "Building..."
	go build -v ./...
	@echo "Running tests..."
	$(MAKE) test
	@echo "All CI checks passed"

# Development workflow
dev: install generate format test
	@echo "Development environment ready"

# Run security scans
sec-scan:
	@echo "🔍 Running security scans..."
	@echo "Installing security tools..."
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "Running govulncheck..."
	$(HOME)/go/bin/govulncheck ./...
	@echo "Running gosec..."
	$(HOME)/go/bin/gosec -exclude=G104,G115,G201,G204,G301,G304,G306 -exclude-dir=internal/interfaces/graphql/generated ./...
	@echo "✅ Security scans completed"

# Quick rebuild and test
quick: generate format test
	@echo "✅ Quick rebuild completed"
