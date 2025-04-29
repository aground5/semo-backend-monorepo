# Build stage
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git build-base

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o semo-server ./cmd/main.go

FROM ghcr.io/wekeepgrowing/ai-sea-executor:main AS executor

# Runtime stage
FROM node:22-alpine

# Add runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Set timezone (can be overridden at runtime)
ENV TZ=UTC

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/semo-server .

# Copy the AI executor from the executor image
COPY --from=executor /app/ai-executor /app/bin/ai-executor

# Create directory for logs with appropriate permissions
RUN mkdir -p /app/logs && chmod -R 777 /app/logs

# Environment variables (can be overridden at runtime)
ENV HTTP_PORT=8080
ENV CONFIG_PATH=/app/configs/file/development.yaml

# Expose the service port
EXPOSE ${HTTP_PORT}

# Health check to verify service is running
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:${HTTP_PORT}/ || exit 1

# Command to run the executable
CMD ["sh", "-c", "./semo-server -c ${CONFIG_PATH}"]