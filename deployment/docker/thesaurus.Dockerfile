# Thesaurus Service Dockerfile
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy shared modules
COPY shared/ ./shared/

# Copy service source
COPY services/thesaurus/ ./services/thesaurus/

# Build the service
WORKDIR /app/services/thesaurus
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create app directory
WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/services/thesaurus/main .

# Expose port
EXPOSE 8001

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8001/health || exit 1

# Run the binary
CMD ["./main"]