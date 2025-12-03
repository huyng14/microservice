# ----------------------------
# Build Stage
# ----------------------------
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy module files first to leverage caching
COPY go.mod go.sum ./
RUN go mod tidy

# Copy the logging service source code
COPY . .

# Build logging service binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o logging .

# ----------------------------
# Runtime Stage
# ----------------------------
FROM alpine:3.20

# Install CA certs (if making TLS outbound calls)
RUN apk add --no-cache ca-certificates && update-ca-certificates

WORKDIR /app

# Copy built binary from build stage
COPY --from=builder /app/logging /app/logging

# Expose gRPC port used by the logging service
EXPOSE 6514

# Start the logging service
CMD ["/app/logging"]