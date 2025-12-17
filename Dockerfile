# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o antigravity-wrapper ./cmd/server

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/antigravity-wrapper /app/antigravity-wrapper

# Create credentials directory
RUN mkdir -p /root/.antigravity

# Expose default port
EXPOSE 3047

# Set environment defaults
ENV ANTIGRAVITY_HOST=0.0.0.0
ENV ANTIGRAVITY_PORT=3047

# Run the server
ENTRYPOINT ["/app/antigravity-wrapper"]