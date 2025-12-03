# Multi-stage build for DA Orchestrator

# Stage 1: Build
FROM golang:1.25.5-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy dago-libs first (needed for replace directive)
COPY dago-libs/ /dago-libs/

# Copy go mod files from dago
COPY dago/go.mod dago/go.sum ./
RUN go mod download

# Copy dago source code
COPY dago/ .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o dago \
    ./cmd/dago

# Stage 2: Runtime
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 dago && \
    adduser -D -u 1000 -G dago dago

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/dago /app/dago

# Change ownership
RUN chown -R dago:dago /app

# Switch to non-root user
USER dago

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/app/dago"]
