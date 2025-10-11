# Standalone ForwardAuth Example

This directory contains a complete working example of deploying traefik-casdoor-auth in standalone ForwardAuth mode.

## Files

- `traefik-static.yml` - Traefik static configuration
- `traefik-dynamic.yml` - Traefik dynamic configuration with ForwardAuth middleware
- `docker-compose.yml` - Complete Docker Compose setup with Traefik and auth service
- `plugin.json` - Casdoor authentication configuration

## Prerequisites

- Docker and Docker Compose installed
- A Casdoor instance (you can use the demo at https://demo.casdoor.com)
- Casdoor application credentials (Client ID and Client Secret)

## Setup

1. **Configure Casdoor credentials:**

   Edit `plugin.json` and update with your Casdoor credentials:
   ```json
   {
       "casdoorEndpoint": "https://your-casdoor-instance.com",
       "casdoorClientId": "your-client-id",
       "casdoorClientSecret": "your-client-secret",
       "casdoorOrganization": "your-org",
       "casdoorApplication": "your-app",
       "pluginEndPoint": "http://localhost:9999"
   }
   ```

2. **Update host configuration (optional):**

   For local testing, add to your `/etc/hosts`:
   ```
   127.0.0.1 app.example.com
   ```

3. **Start the services:**
   ```bash
   docker-compose up -d
   ```

4. **Verify services are running:**
   ```bash
   docker-compose ps
   ```

   You should see:
   - `traefik` - Running on ports 80, 443, and 8080
   - `traefik-casdoor-auth` - Running on port 9999
   - `backend` - Example backend service

## Testing

1. **Test the auth service directly:**
   ```bash
   curl http://localhost:9999/echo
   ```
   Should return information about your request.

2. **Access the protected application:**
   
   Open your browser and navigate to:
   - `http://app.example.com` (or `http://localhost` if you didn't configure hosts)
   
   You should be redirected to Casdoor for authentication.

3. **Access Traefik dashboard:**
   ```bash
   http://localhost:8080/dashboard/
   ```

## Customization

### Protecting Your Own Service

Replace the `backend` service in `docker-compose.yml` with your own service:

```yaml
services:
  my-service:
    image: my-app:latest
    container_name: my-service
    networks:
      - traefik-network
```

Then update `traefik-dynamic.yml`:

```yaml
http:
  routers:
    my-app:
      rule: "Host(`myapp.example.com`)"
      service: my-service
      middlewares:
        - casdoor-auth
  
  services:
    my-service:
      loadBalancer:
        servers:
          - url: "http://my-service:PORT"
```

### Multiple Protected Services

You can protect multiple services by creating additional routers:

```yaml
http:
  routers:
    app1:
      rule: "Host(`app1.example.com`)"
      service: service1
      middlewares:
        - casdoor-auth
    
    app2:
      rule: "Host(`app2.example.com`)"
      service: service2
      middlewares:
        - casdoor-auth
```

### HTTPS/TLS Configuration

For production deployments with HTTPS, update `traefik-static.yml`:

```yaml
entryPoints:
  websecure:
    address: ":443"
    http:
      tls:
        certResolver: letsencrypt

certificatesResolvers:
  letsencrypt:
    acme:
      email: your-email@example.com
      storage: /letsencrypt/acme.json
      httpChallenge:
        entryPoint: web
```

## Troubleshooting

### Check logs:
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f traefik-casdoor-auth
```

### Common Issues

1. **Authentication loop:**
   - Verify `pluginEndPoint` in `plugin.json` matches the actual public URL
   - Check Casdoor redirect URI configuration

2. **Connection refused:**
   - Ensure all services are on the same network
   - Verify service names in configuration match container names

3. **401 Unauthorized:**
   - Check Casdoor credentials in `plugin.json`
   - Verify Casdoor application is properly configured

## Clean Up

To stop and remove all containers:
```bash
docker-compose down
```

To also remove volumes:
```bash
docker-compose down -v
```
