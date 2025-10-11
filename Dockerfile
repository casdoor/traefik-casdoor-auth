# Multi-stage build for traefik-casdoor-auth webhook
FROM golang:1.16-alpine AS builder

# Install ca-certificates for HTTPS during go mod download
RUN apk add --no-cache ca-certificates git

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the webhook binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o webhook ./cmd/webhook/

# Final stage: minimal runtime image using distroless
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/webhook .

# Copy the configuration directory
COPY conf ./conf

# Expose the default port
EXPOSE 9999

# Run the webhook service
ENTRYPOINT ["/app/webhook"]
CMD ["-configFile=/app/conf/plugin.json"]
