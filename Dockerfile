# Use pre-built binary to avoid Railway build timeout issues
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

# Copy pre-built binary
COPY railway-binary ./sql-graph-visualizer
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

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/health || exit 1

# Environment variables
ENV GO_ENV=production
ENV LOG_LEVEL=info

# Default command
CMD ["/app/start.sh"]
