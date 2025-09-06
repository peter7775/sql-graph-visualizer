# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Install gqlgen and ensure all dependencies are available
RUN go install github.com/99designs/gqlgen@latest && \
    go mod tidy

# Generate GraphQL code
RUN gqlgen generate

# Build the application
ARG VERSION=dev
ARG BUILD_DATE=unknown

# Build binary with optimization flags
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.BuildDate=${BUILD_DATE}" \
    -o sql-graph-visualizer \
    ./cmd/main.go

# Final stage
FROM alpine:3.20

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    mysql-client \
    curl \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# Set working directory
WORKDIR /app

# Copy built binary from builder stage
COPY --from=builder /app/sql-graph-visualizer .

# Copy configuration files
COPY --from=builder /app/config ./config

# Copy static files if any
COPY --from=builder /app/internal/interfaces/web ./internal/interfaces/web

# Create directories for logs and data
RUN mkdir -p /app/logs /app/data && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 3000 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:3000/config || exit 1

# Environment variables
ENV GO_ENV=production
ENV LOG_LEVEL=info

# Default command
CMD ["./sql-graph-visualizer"]
