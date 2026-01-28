# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /node-agent .

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    docker-cli \
    tzdata

# Create non-root user
RUN adduser -D -H -s /sbin/nologin agent

# Copy binary from builder
COPY --from=builder /node-agent /usr/local/bin/node-agent

# Create config directory
RUN mkdir -p /etc/sing-box && chown agent:agent /etc/sing-box

# Expose port
EXPOSE 9090

# Run as root to access Docker socket (can be changed if socket permissions allow)
USER root

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:9090/health -O /dev/null || exit 1

ENTRYPOINT ["/usr/local/bin/node-agent"]
