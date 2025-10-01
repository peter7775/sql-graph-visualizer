# Multi-stage build to reduce final image size
# Build date: 2025-10-01-14:32 - Cloud build approach
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o sql-graph-visualizer cmd/main.go

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

# Copy binary from builder stage
COPY --from=builder /app/sql-graph-visualizer ./sql-graph-visualizer
RUN chmod +x ./sql-graph-visualizer

# Copy go.mod so findProjectRoot() can locate project root
COPY go.mod ./go.mod

# Copy configuration files
COPY config ./config

# Copy static files if any  
COPY internal/interfaces/web ./internal/interfaces/web

# Copy init SQL for optional DB bootstrap
COPY railway-mysql-init.sql ./railway-mysql-init.sql

# Copy entrypoint
COPY start.sh ./start.sh
RUN chmod +x ./start.sh

# Create directories for logs and data
RUN mkdir -p /app/logs /app/data && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 3000 8080

# Railway uses its own healthcheck system, so Docker HEALTHCHECK is not needed

# Environment variables
ENV GO_ENV=production
ENV LOG_LEVEL=info

# Default command
CMD ["/app/start.sh"]
