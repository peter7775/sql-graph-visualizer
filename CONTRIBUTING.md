## Contributing to MySQL Graph Visualizer

Thank you for your interest in contributing to MySQL Graph Visualizer! This document provides guidelines and information for contributors.

## 🚀 Quick Start

1. **Fork** the repository on GitHub
2. **Clone** your fork locally
3. **Install** dependencies: `go mod tidy`
4. **Start** Neo4j: `docker-compose up -d neo4j-test`
5. **Run** tests: `go test ./...`
6. **Start** the application: `go run cmd/main.go`

## 📋 How to Contribute

### Reporting Bugs

Before creating bug reports, please check existing issues to avoid duplicates. When creating a bug report, include:

- **Description**: Clear description of what the bug is
- **Steps to reproduce**: Detailed steps to reproduce the behavior
- **Expected behavior**: What you expected to happen
- **Environment**: OS, Go version, database versions
- **Screenshots**: If applicable
- **Additional context**: Any other relevant information

### Suggesting Features

Feature requests are welcome! Please:

- **Check existing issues** to avoid duplicates
- **Describe the feature** in detail
- **Explain the motivation** - why is this feature needed?
- **Provide examples** of how it would be used

### Code Contributions

1. **Fork and Clone**

   ```bash
   git clone https://github.com/YOUR_USERNAME/mysql-graph-visualizer.git
   cd mysql-graph-visualizer
   ```

2. **Create a Branch**

   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/issue-number
   ```

3. **Make Changes**
   - Follow the coding standards below
   - Write tests for new functionality
   - Update documentation as needed

4. **Test Your Changes**

   ```bash
   go test ./...
   go build ./cmd/main.go
   ```

5. **Commit and Push**

   ```bash
   git add .
   git commit -m "feat: add your feature description"
   git push origin your-branch-name
   ```

6. **Create Pull Request**
   - Use the PR template
   - Link related issues
   - Provide clear description

## 🏗️ Development Setup

### Prerequisites

- Go 1.22.5+
- MySQL 8.0+
- Neo4j 4.4+
- Docker (for Neo4j)

### Local Development

1. **Environment Setup**

   ```bash
   # Clone the project
   git clone https://github.com/YOUR_USERNAME/mysql-graph-visualizer.git
   cd mysql-graph-visualizer

   # Install dependencies
   go mod tidy

   # Start Neo4j
   docker-compose up -d neo4j-test

   # Copy and configure
   cp config/config-test.yml config/config.yml
   # Edit config.yml with your MySQL credentials
   ```

2. **Running the Application**

   ```bash
   # Development mode with debug logging
   LOG_LEVEL=debug go run cmd/main.go

   # Production build
   go build -o mysql-graph-visualizer cmd/main.go
   ./mysql-graph-visualizer
   ```

3. **Running Tests**

   ```bash
   # Run all tests
   go test ./...

   # Run with coverage
   go test -cover ./...

   # Run specific package
   go test ./internal/domain/...
   ```

## 📝 Coding Standards

### Go Style Guide

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting: `go fmt ./...`
- Use `golint` for linting: `golint ./...`
- Use `go vet` for analysis: `go vet ./...`

### Architecture Principles

- **Domain Driven Design**: Keep domain logic pure
- **Ports & Adapters**: Use interfaces for external dependencies
- **Single Responsibility**: Each component has one responsibility
- **Dependency Injection**: Inject dependencies via constructors

### Code Organization

```
internal/
├── application/     # Use cases and application services
│   ├── ports/      # Interface definitions
│   └── services/   # Application logic
├── domain/         # Core business logic
│   ├── entities/   # Domain entities
│   ├── aggregates/ # Domain aggregates
│   └── models/     # Domain models
├── infrastructure/ # External concerns
│   └── persistence/ # Database repositories
└── interfaces/     # API and web interfaces
```

### Naming Conventions

- **Files**: `snake_case.go`
- **Packages**: `lowercase`
- **Types**: `PascalCase`
- **Functions/Methods**: `PascalCase` (exported), `camelCase` (unexported)
- **Constants**: `UPPER_SNAKE_CASE`
- **Variables**: `camelCase`

### Documentation

- All exported functions/types must have comments
- Use GoDoc format
- Include examples for complex functions
- Update README.md for user-facing changes

## 🧪 Testing Guidelines

### Test Structure

- Unit tests: `*_test.go` alongside source files
- Integration tests: `tests/integration/`
- Test data: `testdata/`

### Test Naming

```go
func TestServiceName_MethodName_ExpectedBehavior(t *testing.T)
func TestUserService_CreateUser_ReturnsErrorWhenEmailExists(t *testing.T)
```

### API Documentation

- Use OpenAPI/Swagger for REST APIs
- Use GraphQL introspection for GraphQL schema
- Include examples in documentation

### Code Documentation

- GoDoc for all exported symbols
- Inline comments for complex logic
- Architecture decision records (ADRs) for major decisions

## 🏷️ Issue Labels

Understanding our issue labels:

- `good first issue` - Perfect for newcomers
- `help wanted` - Community help needed
- `bug` - Something isn't working
- `enhancement` - New feature or improvement
- `documentation` - Documentation related
- `question` - Further information requested
- `priority/high` - High priority issues
- `area/backend` - Backend related
- `area/frontend` - Frontend/UI related
- `area/database` - Database related
