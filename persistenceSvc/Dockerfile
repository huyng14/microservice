# ----------------------------
# Build Stage
# ----------------------------
FROM golang:1.25-alpine AS builder

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Work directory inside the builder
WORKDIR /app

# Copy go.mod and go.sum first to enable caching
COPY go.mod go.sum ./
RUN go mod tidy

# Copy the entire persistenceSvc source code
COPY . .

# Build the persistence service binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o persistence .

# ----------------------------
# Runtime Stage
# ----------------------------
FROM alpine:3.20

# Install CA certificates for TLS (database, internal calls, etc.)
RUN apk add --no-cache ca-certificates && update-ca-certificates

# Work directory inside the runtime container
WORKDIR /app

# Copy compiled binary from builder image
COPY --from=builder /app/persistence /app/persistence

# Expose gRPC port
EXPOSE 9000

# Start persistence service
CMD ["/app/persistence"]