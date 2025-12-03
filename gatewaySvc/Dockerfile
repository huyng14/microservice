# Build stage
FROM golang:1.22-alpine AS builder

# Work in /app at the module root
WORKDIR /app

# Install build dependencies (if needed) and CA certificates
RUN apk add --no-cache ca-certificates git

# Go module files first to leverage Docker layer caching
# NOTE: this expects go.mod and go.sum to be at the repo root, not in gatewaySvc/
COPY go.mod go.sum ./
RUN go mod download

# Copy only the gateway service source into the image
COPY gatewaySvc ./gatewaySvc

# Build the gateway binary from the gatewaySvc package
WORKDIR /app/gatewaySvc
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/gateway .

# Runtime stage
FROM alpine:3.20

# Install CA certificates for outbound TLS connections
RUN apk add --no-cache ca-certificates && update-ca-certificates

WORKDIR /app

# Copy compiled binary from builder stage
COPY --from=builder /app/gateway /app/gateway

# Environment variables for downstream gRPC services
ENV PERSISTENCE_GRPC_ADDR=persistenceSvc:9000 \
    LOGGING_GRPC_ADDR=loggingSvc:6514 \
    PORT=8080

# HTTP port exposed by the gateway
EXPOSE 8080

# Run the gateway service
CMD ["/app/gateway"]
