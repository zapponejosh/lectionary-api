# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies for SQLite
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files first (better layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with CGO enabled (required for SQLite)
# -ldflags "-s -w" strips debug info for smaller binary
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags "-s -w" -o /app/api ./cmd/api

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
  ca-certificates \
  tzdata \
  sqlite

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/api .

# Copy migrations (needed for initial setup)
COPY migrations/ ./migrations/

# Create data directory
RUN mkdir -p /data

# Non-root user for security
RUN adduser -D -H -h /app appuser
RUN chown -R appuser:appuser /app /data
USER appuser

# Environment defaults
ENV PORT=8080
ENV ENV=production
ENV DATABASE_PATH=/data/lectionary.db
ENV LOG_FORMAT=json
ENV LOG_LEVEL=info

EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/api"]