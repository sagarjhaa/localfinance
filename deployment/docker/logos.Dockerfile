# Logos Document Processing Service Dockerfile
FROM golang:1.21-alpine AS builder

# Install build dependencies for PDF processing
RUN apk add --no-cache git gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy shared modules
COPY shared/ ./shared/

# Copy service source
COPY services/logos/ ./services/logos/

# Build the service
WORKDIR /app/services/logos
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

# Install runtime dependencies for document processing
RUN apk --no-cache add ca-certificates libc6-compat

# Create app directory
WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/services/logos/main .

# Create directories for file processing
RUN mkdir -p /tmp/uploads /tmp/processing

# Expose port
EXPOSE 8003

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8003/health || exit 1

# Run the binary
CMD ["./main"]