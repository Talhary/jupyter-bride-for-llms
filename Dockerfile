# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build statically linked binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o jupyter-bridge .

# Final minimal production stage (~6.8MB)
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/jupyter-bridge /app/jupyter-bridge

# Expose HTTP / SSE port for remote MCP mode
EXPOSE 8080

# Run as non-root user for security
USER 1000:1000

ENTRYPOINT ["/app/jupyter-bridge"]
