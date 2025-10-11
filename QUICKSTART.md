# Quick Start Guide - Standalone ForwardAuth Mode

This guide helps you get traefik-casdoor-auth up and running in standalone mode in just a few minutes.

## Prerequisites

- Docker and Docker Compose installed
- A Casdoor instance (use https://demo.casdoor.com for testing)
- 5 minutes of your time

## 5-Minute Setup

### Step 1: Get Your Casdoor Credentials

If you don't have a Casdoor application yet:

1. Go to https://demo.casdoor.com (or your Casdoor instance)
2. Sign in (create account if needed)
3. Navigate to Applications
4. Create a new application or use an existing one
5. Note down:
   - Client ID
   - Client Secret
   - Organization name
   - Application name

### Step 2: Clone and Configure

```bash
# Clone the repository
git clone https://github.com/casdoor/traefik-casdoor-auth.git
cd traefik-casdoor-auth

# Navigate to the standalone example
cd examples/standalone

# Edit the configuration file
nano plugin.json  # or use your favorite editor
```

Update `plugin.json` with your credentials:
```json
{
    "casdoorEndpoint": "https://demo.casdoor.com",
    "casdoorClientId": "YOUR_CLIENT_ID_HERE",
    "casdoorClientSecret": "YOUR_CLIENT_SECRET_HERE",
    "casdoorOrganization": "built-in",
    "casdoorApplication": "app-built-in",
    "pluginEndPoint": "http://localhost:9999"
}
```

### Step 3: Start Everything

```bash
# Start all services
docker-compose up -d

# Verify services are running
docker-compose ps
```

You should see:
- ✓ traefik (ports 80, 443, 8080)
- ✓ traefik-casdoor-auth (port 9999)
- ✓ backend (protected service)

### Step 4: Test It

Open your browser and go to:
```
http://localhost
```

You should be:
1. Redirected to Casdoor login page
2. Asked to sign in
3. Redirected back to see the protected service

🎉 **That's it!** Your authentication is working!

## What Just Happened?

1. Traefik received your request
2. ForwardAuth middleware sent request to traefik-casdoor-auth
3. You were redirected to Casdoor for authentication
4. After login, Casdoor redirected you back
5. Your session was established
6. You got access to the protected service

## Next Steps

### View Traefik Dashboard

```
http://localhost:8080/dashboard/
```

### Check Logs

```bash
# View all logs
docker-compose logs -f

# View auth service logs only
docker-compose logs -f traefik-casdoor-auth
```

### Protect Your Own Service

Edit `docker-compose.yml` and replace the `backend` service with your own:

```yaml
services:
  my-app:
    image: my-app:latest
    networks:
      - traefik-network
```

Then update `traefik-dynamic.yml` to point to your service.

### Deploy to Production

For production deployment:

1. **Use HTTPS**: Configure TLS/SSL certificates
2. **Use Secrets**: Store credentials securely (not in plain text)
3. **Scale**: Run multiple instances for high availability
4. **Monitor**: Set up logging and monitoring

See the full [README.md](README.md) for production deployment guides.

## Troubleshooting

### Service won't start
```bash
# Check logs
docker-compose logs

# Restart services
docker-compose restart
```

### Can't access localhost
- Try `http://127.0.0.1` instead
- Check if ports 80, 8080, 9999 are free
- Verify Docker is running

### Authentication loop
- Verify `pluginEndPoint` matches your actual URL
- Check Casdoor redirect URI configuration
- Clear browser cookies and try again

### Still having issues?
1. Check the [examples/standalone/README.md](examples/standalone/README.md)
2. See [MIGRATION.md](MIGRATION.md) for detailed configuration
3. Open an issue on GitHub with your error logs

## Clean Up

When you're done testing:

```bash
# Stop services
docker-compose down

# Remove everything including volumes
docker-compose down -v
```

## Production Deployment Options

Ready for production? Choose your platform:

- **Docker**: See [docker-compose.standalone.yml](docker-compose.standalone.yml)
- **Kubernetes**: See [k8s/README.md](k8s/README.md)
- **Manual Build**: See [BUILD.md](BUILD.md)

## Learn More

- [Full README](README.md) - Complete documentation
- [Migration Guide](MIGRATION.md) - Migrate from plugin mode
- [Build Guide](BUILD.md) - Build from source
- [Kubernetes Guide](k8s/README.md) - K8s deployment
