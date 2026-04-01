FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY shared/ /shared/
COPY services/thesaurus/ .
RUN go mod edit -replace github.com/sagarjhaa/localfinance/shared=/shared
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /service .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates curl
COPY --from=builder /service /service
HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD curl -sf http://localhost:${PORT:-8001}/health || exit 1
ENTRYPOINT ["/service"]
