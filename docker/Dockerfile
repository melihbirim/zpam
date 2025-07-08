# Build stage
FROM golang:1.24-alpine AS builder

# Install necessary packages
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o zpam .

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S zpam && \
    adduser -u 1001 -S zpam -G zpam

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/zpam .

# Copy configuration files
COPY --from=builder /app/config*.yaml ./
COPY --from=builder /app/custom_rules*.yml ./

# Create directories for data
RUN mkdir -p /app/data /app/logs /app/models && \
    chown -R zpam:zpam /app

# Switch to non-root user
USER zpam

# Expose milter port
EXPOSE 7357

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD timeout 5s /app/zpam --help > /dev/null 2>&1 || exit 1

# Default command
CMD ["./zpam", "milter", "--config", "config.yaml"] 