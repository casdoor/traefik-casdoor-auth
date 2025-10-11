# Build Instructions

This document provides instructions for building the traefik-casdoor-auth webhook service.

## Prerequisites

- Go 1.16 or later
- Docker (for containerized builds)
- Git

## Building from Source

### Building the Binary

```bash
# Clone the repository
git clone https://github.com/casdoor/traefik-casdoor-auth.git
cd traefik-casdoor-auth

# Download dependencies
go mod download

# Build the webhook binary
go build -o webhook ./cmd/webhook/

# Run the webhook
./webhook -configFile="conf/plugin.json"
```

### Cross-Compilation

Build for different platforms:

```bash
# Linux (amd64)
GOOS=linux GOARCH=amd64 go build -o webhook-linux-amd64 ./cmd/webhook/

# Linux (arm64)
GOOS=linux GOARCH=arm64 go build -o webhook-linux-arm64 ./cmd/webhook/

# Windows (amd64)
GOOS=windows GOARCH=amd64 go build -o webhook-windows-amd64.exe ./cmd/webhook/

# macOS (amd64)
GOOS=darwin GOARCH=amd64 go build -o webhook-darwin-amd64 ./cmd/webhook/

# macOS (arm64 - M1/M2)
GOOS=darwin GOARCH=arm64 go build -o webhook-darwin-arm64 ./cmd/webhook/
```

## Building Docker Image

### Standard Build

```bash
docker build -t traefik-casdoor-auth:latest .
```

### Multi-Architecture Build

Using Docker Buildx for multi-platform images:

```bash
# Create a new builder instance
docker buildx create --name multiarch --use

# Build for multiple architectures
docker buildx build \
  --platform linux/amd64,linux/arm64,linux/arm/v7 \
  -t traefik-casdoor-auth:latest \
  --push \
  .
```

### Build with Custom Tags

```bash
# Build with version tag
docker build -t traefik-casdoor-auth:v1.0.0 .

# Build and tag as latest
docker build -t traefik-casdoor-auth:latest .

# Build with multiple tags
docker build \
  -t traefik-casdoor-auth:latest \
  -t traefik-casdoor-auth:v1.0.0 \
  .
```

## Development Build

For development, you can build with debugging symbols:

```bash
go build -gcflags="all=-N -l" -o webhook-debug ./cmd/webhook/
```

## Static Binary Build

For maximum compatibility and smaller images:

```bash
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o webhook ./cmd/webhook/
```

## Testing the Build

After building, test the webhook:

```bash
# Start the webhook
./webhook -configFile="conf/plugin.json"

# In another terminal, test the echo endpoint
curl http://localhost:9999/echo
```

## CI/CD

The project includes GitHub Actions workflows for automated building:

- `.github/workflows/ci.yml` - Runs tests on push/PR
- `.github/workflows/docker.yml` - Builds and publishes Docker images

Docker images are automatically published to GitHub Container Registry on:
- Push to master/main branch (tagged as `latest`)
- Git tags (tagged with version number)

## Troubleshooting

### Build Fails with Missing Dependencies

```bash
go mod tidy
go mod download
```

### Docker Build Network Issues

If you encounter network issues during Docker build, try:

```bash
# Use BuildKit with inline cache
DOCKER_BUILDKIT=1 docker build --network=host -t traefik-casdoor-auth .
```

### Permission Denied on Binary

```bash
chmod +x webhook
```
