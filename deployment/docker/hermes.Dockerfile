# Hermes Frontend & API Gateway Dockerfile
FROM node:18-alpine AS frontend-builder

# Set working directory for frontend build
WORKDIR /app/frontend

# Copy package files (will be created)
# COPY services/hermes/frontend/package*.json ./
# RUN npm install

# Copy frontend source
# COPY services/hermes/frontend/ ./
# RUN npm run build

# Go backend builder
FROM golang:1.21-alpine AS backend-builder

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
COPY services/hermes/ ./services/hermes/

# Build the service
WORKDIR /app/services/hermes
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

# Install ca-certificates and serve static files
RUN apk --no-cache add ca-certificates

# Create app directory
WORKDIR /root/

# Copy backend binary
COPY --from=backend-builder /app/services/hermes/main .

# Copy frontend build (when available)
# COPY --from=frontend-builder /app/frontend/build ./frontend/build

# Create frontend directory structure for now
RUN mkdir -p ./frontend/build
RUN echo '<!DOCTYPE html><html><head><title>LocalFinance</title></head><body><h1>LocalFinance</h1><p>Frontend coming soon...</p></body></html>' > ./frontend/build/index.html

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000/health || exit 1

# Run the binary
CMD ["./main"]