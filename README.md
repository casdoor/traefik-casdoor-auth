# Traefik Casdoor Auth Plugin

<p align="center">
  <a href="#badge">
    <img alt="semantic-release" src="https://img.shields.io/badge/%20%20%F0%9F%93%A6%F0%9F%9A%80-semantic--release-e10079.svg">
  </a>
  <a href="https://github.com/casdoor/traefik-casdoor-auth/actions/workflows/ci.yml">
    <img alt="GitHub Workflow Status (branch)" src="https://img.shields.io/github/actions/workflow/status/casdoor/traefik-casdoor-auth/ci.yml?branch=master">
  </a>
  <a href="https://github.com/casdoor/traefik-casdoor-auth/releases/latest">
    <img alt="GitHub Release" src="https://img.shields.io/github/v/release/casdoor/traefik-casdoor-auth.svg">
  </a>
</p>

<p align="center">
  <a href="https://goreportcard.com/report/github.com/casdoor/traefik-casdoor-auth">
    <img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/casdoor/traefik-casdoor-auth?style=flat-square">
  </a>
  <a href="https://github.com/casdoor/traefik-casdoor-auth/blob/master/LICENSE">
    <img src="https://img.shields.io/github/license/casdoor/traefik-casdoor-auth?style=flat-square" alt="license">
  </a>
  <a href="https://github.com/casdoor/traefik-casdoor-auth/issues">
    <img alt="GitHub issues" src="https://img.shields.io/github/issues/casdoor/traefik-casdoor-auth?style=flat-square">
  </a>
  <a href="#">
    <img alt="GitHub stars" src="https://img.shields.io/github/stars/casdoor/traefik-casdoor-auth?style=flat-square">
  </a>
  <a href="https://github.com/casdoor/traefik-casdoor-auth/network">
    <img alt="GitHub forks" src="https://img.shields.io/github/forks/casdoor/traefik-casdoor-auth?style=flat-square">
  </a>
  <a href="https://discord.gg/5rPsrAzK7S">
    <img alt="Casdoor" src="https://img.shields.io/discord/1022748306096537660?style=flat-square&logo=discord&label=discord&color=5865F2">
  </a>
</p>

A powerful Traefik middleware plugin that integrates [Casdoor](https://casdoor.org/) authentication to protect your HTTP services. This solution provides seamless SSO (Single Sign-On) capabilities without requiring any changes to your backend services.

## 📋 Table of Contents

- [Features](#-features)
- [Architecture](#-architecture)
- [Installation](#-installation)
- [Quick Start](#-quick-start)
- [Configuration](#-configuration)
- [How It Works](#-how-it-works)
- [Docker Deployment](#-docker-deployment)
- [Development](#-development)
- [Contributing](#-contributing)
- [License](#-license)

## ✨ Features

- **Zero Backend Changes**: Add authentication to any HTTP service without modifying your application code
- **Seamless SSO**: Integrate with Casdoor for centralized authentication and user management
- **Traefik Middleware**: Implemented as a native Traefik plugin for easy integration
- **Session Management**: Automatic session handling with secure cookies
- **OAuth 2.0 Flow**: Complete OAuth 2.0 implementation with PKCE support
- **Request Forwarding**: Transparent request modification and forwarding
- **Stateless Architecture**: Webhook-based design for scalability

## 🏗 Architecture

This solution consists of two main components:

1. **Traefik Plugin**: A middleware that intercepts HTTP requests and forwards them to the webhook for authentication decisions
2. **Authentication Webhook**: A service that validates user sessions, handles OAuth flows, and instructs the plugin how to process requests

### Component Interaction

```
┌─────────┐      ┌──────────────┐      ┌──────────┐      ┌─────────┐
│ Client  │─────▶│   Traefik    │─────▶│ Webhook  │─────▶│ Casdoor │
│         │      │   Plugin     │      │  Service │      │  Server │
└─────────┘      └──────────────┘      └──────────┘      └─────────┘
                        │                     │
                        └─────────────────────┘
                         Authentication Flow
```

## 📦 Installation

### Using Docker (Recommended)

A pre-built webhook image is available for easy deployment:

```bash
docker pull ghcr.io/lostb1t/traefik-casdoor-auth:latest
```

For more details, visit: https://github.com/lostb1t/traefik-casdoor-auth

### From Source

**Prerequisites:**
- Go 1.16 or higher
- Traefik v2.x
- Docker (for running example services)
- A running [Casdoor](https://casdoor.org/) instance

Clone the repository:

```bash
git clone https://github.com/casdoor/traefik-casdoor-auth.git
cd traefik-casdoor-auth
```

## 🚀 Quick Start

### Step 1: Configure Casdoor

1. Access your Casdoor admin panel
2. Create a new application for Traefik authentication
3. Note down the following details:
   - **Client ID**
   - **Client Secret**
   - **Organization Name**
   - **Application Name**

For detailed instructions, see [Casdoor Application Configuration](https://casdoor.org/docs/application/config/).

### Step 2: Configure Traefik Static Configuration

Create or update your `traefik.yml`:

```yaml
entryPoints:
  web:
    address: ":80"

experimental:
  localPlugins:
    example:
      moduleName: github.com/casdoor/plugindemo

api:
  insecure: true

providers:
  file:
    filename: dev.yml
```

**Note**: The `moduleName` must match the path relative to `plugins-local/src/` and the module name in `plugins-local/src/github.com/casdoor/plugindemo/.traefik.yml`.

### Step 3: Configure Traefik Dynamic Configuration

Create `dev.yml`:

```yaml
http:
  routers:
    my-router:
      rule: host(`webhook.domain.local`)
      service: service-foo
      entryPoints:
        - web
      middlewares:
        - my-plugin

  services:
    service-foo:
      loadBalancer:
        servers:
          - url: http://127.0.0.1:5000

  middlewares:
    my-plugin:
      plugin:
        example:
          multationWebhook: "http://webhook.domain.local:9999/auth"
```

### Step 4: Configure the Webhook

Create or update `conf/plugin.json`:

```json
{
    "casdoorEndpoint": "http://webhook.domain.local:8000",
    "casdoorClientId": "YOUR_CLIENT_ID",
    "casdoorClientSecret": "YOUR_CLIENT_SECRET",
    "casdoorOrganization": "YOUR_ORGANIZATION",
    "casdoorApplication": "YOUR_APPLICATION",
    "pluginEndPoint": "http://webhook.domain.local:9999"
}
```

**Configuration Parameters:**

| Parameter | Description |
|-----------|-------------|
| `casdoorEndpoint` | URL of your Casdoor server |
| `casdoorClientId` | Client ID from Casdoor application |
| `casdoorClientSecret` | Client secret from Casdoor application |
| `casdoorOrganization` | Organization name in Casdoor |
| `casdoorApplication` | Application name in Casdoor |
| `pluginEndPoint` | URL where the webhook service is accessible |

### Step 5: Update Hosts File

Add the following entry to your hosts file (`/etc/hosts` on Linux/Mac, `C:\Windows\System32\drivers\etc\hosts` on Windows):

```
127.0.0.1    webhook.domain.local
```

### Step 6: Start Services

**Start the example service:**

```bash
docker compose up -d
```

This starts a "whoami" container on port 5000 - a simple HTTP service that echoes request information.

**Start Traefik:**

```bash
sudo traefik --configFile="traefik.yml" --log.level=DEBUG
```

**Start the webhook service:**

```bash
go run cmd/webhook/main.go -configFile="conf/plugin.json"
```

### Step 7: Test the Setup

Visit http://webhook.domain.local in your browser.

- **First visit**: You'll be redirected to Casdoor for authentication
- **After login**: You'll be redirected back and see the "whoami" service output

## ⚙️ Configuration

### Traefik Plugin Configuration

The plugin accepts the following parameter:

- `multationWebhook`: The URL of the authentication webhook endpoint

### Webhook Configuration

All webhook configuration is stored in `conf/plugin.json`. See the [Quick Start](#step-4-configure-the-webhook) section for available parameters.

## 🔄 How It Works

### Authentication Flow

1. **Initial Request**: Client requests a protected resource
2. **Plugin Intercept**: Traefik plugin intercepts the request and forwards it to the webhook
3. **Session Check**: Webhook checks for a valid authentication cookie
4. **Redirect to Login**: If no valid session exists, webhook returns a 302 redirect to Casdoor
5. **User Authentication**: User logs in via Casdoor
6. **OAuth Callback**: Casdoor redirects to the webhook's callback handler
7. **Token Exchange**: Webhook exchanges the authorization code for an access token
8. **Cookie Creation**: Webhook sets a secure authentication cookie
9. **Original Request Replay**: User is redirected to the original URL
10. **Request Modification**: Plugin modifies the request based on webhook instructions and forwards to the backend service

### Response Handling

- **2xx Response from Webhook**: Request is modified according to webhook instructions and forwarded to the backend
- **Non-2xx Response**: Request is blocked, and the webhook's response (including status code and body) is returned to the client

## 🐳 Docker Deployment

For production deployments, you can use Docker Compose. Here's an example configuration:

```yaml
version: '3'

services:
  traefik:
    image: traefik:v2.9
    command:
      - "--configFile=/etc/traefik/traefik.yml"
      - "--log.level=INFO"
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./traefik.yml:/etc/traefik/traefik.yml
      - ./dev.yml:/etc/traefik/dev.yml
      - ./plugins-local:/plugins-local

  casdoor-auth:
    image: ghcr.io/lostb1t/traefik-casdoor-auth:latest
    environment:
      - CONFIG_FILE=/config/plugin.json
    volumes:
      - ./conf:/config
    ports:
      - "9999:9999"
```

## 🛠 Development

### Running Tests

Run the plugin tests:

```bash
cd plugins-local/src/github.com/casdoor/plugindemo
go test -v ./...
```

### Project Structure

```
.
├── cmd/
│   └── webhook/           # Webhook service main package
├── internal/
│   ├── config/           # Configuration handling
│   ├── handler/          # HTTP handlers
│   └── httpstate/        # Session state management
├── plugins-local/
│   └── src/
│       └── github.com/
│           └── casdoor/
│               └── plugindemo/  # Traefik plugin implementation
├── conf/                 # Configuration files
├── .github/
│   └── workflows/        # CI/CD workflows
└── README.md
```

### Building the Webhook

```bash
go build -o webhook cmd/webhook/main.go
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes using [Conventional Commits](https://www.conventionalcommits.org/)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

### Commit Convention

This project uses [semantic-release](https://github.com/semantic-release/semantic-release) for automated version management and package publishing. Please use the following commit message format:

- `feat:` - A new feature (triggers minor version bump)
- `fix:` - A bug fix (triggers patch version bump)
- `docs:` - Documentation changes
- `style:` - Code style changes (formatting, etc.)
- `refactor:` - Code refactoring
- `perf:` - Performance improvements
- `test:` - Adding or updating tests
- `chore:` - Maintenance tasks

Example: `feat: add support for custom claim mapping`

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🌟 Acknowledgments

- [Traefik](https://traefik.io/) - Cloud Native Application Proxy
- [Casdoor](https://casdoor.org/) - UI-first Identity Access Management (IAM) / Single-Sign-On (SSO) platform

## 📞 Support

- 📫 [GitHub Issues](https://github.com/casdoor/traefik-casdoor-auth/issues)
- 💬 [Discord Community](https://discord.gg/5rPsrAzK7S)
- 📖 [Casdoor Documentation](https://casdoor.org/docs/overview)

---

Made with ❤️ by the [Casdoor](https://casdoor.org/) team

