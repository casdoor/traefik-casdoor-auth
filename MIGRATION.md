# Migration Guide: Plugin Mode to Standalone ForwardAuth

This guide helps you migrate from the experimental Traefik plugin mode to the production-ready standalone ForwardAuth mode.

## Why Migrate?

- **Production Ready**: Standalone mode doesn't rely on Traefik's experimental plugin system
- **Kubernetes Native**: Easier deployment in Kubernetes environments
- **Better Scalability**: Can be scaled independently of Traefik
- **Easier Updates**: Update the auth service without restarting Traefik
- **Better Monitoring**: Independent service means independent monitoring and logging

## Prerequisites

Before migrating, ensure you have:
- Access to your Traefik configuration
- Docker or Kubernetes environment
- Your Casdoor credentials from `conf/plugin.json`

## Migration Steps

### Step 1: Deploy the Standalone Service

Choose your deployment method:

#### Option A: Docker
```bash
docker run -d \
  -p 9999:9999 \
  -v $(pwd)/conf/plugin.json:/app/conf/plugin.json:ro \
  --name traefik-casdoor-auth \
  --network traefik-network \
  ghcr.io/casdoor/traefik-casdoor-auth:latest
```

#### Option B: Docker Compose
```bash
docker-compose -f docker-compose.standalone.yml up -d
```

#### Option C: Kubernetes
```bash
kubectl apply -f k8s/deployment.yaml
```

### Step 2: Update Traefik Static Configuration

**Before (Plugin Mode):**
```yaml
experimental:
  localPlugins:
    example:
      moduleName: github.com/casdoor/plugindemo
```

**After (Standalone Mode):**
Remove the experimental plugin configuration entirely. The static configuration becomes simpler:
```yaml
entryPoints:
  web:
    address: ":80"
api:
  insecure: true
providers:
  file:
    filename: dynamic.yml
```

### Step 3: Update Traefik Dynamic Configuration

**Before (Plugin Mode):**
```yaml
http:
  routers:
    my-router:
      rule: Host(`webhook.domain.local`)
      service: service-foo
      entryPoints:
        - web
      middlewares:
        - my-plugin

  middlewares:
    my-plugin:
      plugin:
        example:
          multationWebhook: "http://webhook.domain.local:9999/auth"
```

**After (Standalone Mode):**
```yaml
http:
  routers:
    my-router:
      rule: Host(`webhook.domain.local`)
      service: service-foo
      entryPoints:
        - web
      middlewares:
        - casdoor-auth

  middlewares:
    casdoor-auth:
      forwardAuth:
        address: http://traefik-casdoor-auth:9999/auth
        trustForwardHeader: true
```

### Step 4: Verify the Migration

1. **Check the service is running:**
   ```bash
   curl http://localhost:9999/echo
   ```
   You should receive a response indicating the service is healthy.

2. **Test authentication:**
   Access your protected service through Traefik. You should be redirected to Casdoor for authentication.

3. **Check logs:**
   ```bash
   # Docker
   docker logs traefik-casdoor-auth
   
   # Kubernetes
   kubectl logs -n traefik-casdoor-auth -l app=traefik-casdoor-auth
   ```

### Step 5: Remove Old Plugin Files (Optional)

Once you've verified the standalone mode works, you can optionally remove the plugin files:
```bash
rm -rf plugins-local/
```

**Note:** Keep these files if you want to maintain compatibility with the plugin mode for testing or rollback purposes.

## Kubernetes-Specific Migration

### Using Traefik CRDs

**Before (Plugin Mode):**
```yaml
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: my-plugin
spec:
  plugin:
    example:
      multationWebhook: "http://webhook.domain.local:9999/auth"
```

**After (Standalone Mode):**
```yaml
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: casdoor-auth
spec:
  forwardAuth:
    address: http://traefik-casdoor-auth.traefik-casdoor-auth.svc.cluster.local:9999/auth
    trustForwardHeader: true
```

## Configuration Differences

The configuration file (`conf/plugin.json`) remains the same between plugin and standalone modes:

```json
{
    "casdoorEndpoint": "https://your-casdoor-server.com",
    "casdoorClientId": "your-client-id",
    "casdoorClientSecret": "your-client-secret",
    "casdoorOrganization": "YourOrganization",
    "casdoorApplication": "YourApplication",
    "pluginEndPoint": "https://auth.yourdomain.com"
}
```

**Important:** Make sure `pluginEndPoint` matches the public URL where the standalone service is accessible.

## Rollback Plan

If you need to rollback to plugin mode:

1. Restore the original Traefik configuration
2. Stop the standalone service:
   ```bash
   docker stop traefik-casdoor-auth
   # or
   kubectl delete -f k8s/deployment.yaml
   ```
3. Restart Traefik to reload the plugin configuration

## Troubleshooting

### Issue: Authentication not working after migration

**Solution:** 
- Verify the ForwardAuth address is correct
- Check network connectivity between Traefik and the auth service
- Ensure `trustForwardHeader: true` is set

### Issue: Service unavailable

**Solution:**
- Check if the service is running: `docker ps` or `kubectl get pods`
- Verify port 9999 is accessible
- Check logs for errors

### Issue: Redirect loop

**Solution:**
- Verify `pluginEndPoint` in configuration matches the actual public URL
- Check that the callback URL is accessible
- Ensure cookies are being set correctly

## Performance Considerations

The standalone mode offers several performance benefits:

- **Horizontal Scaling**: Scale the auth service independently based on load
- **Connection Pooling**: Better connection management between Traefik and auth service
- **Caching**: State management remains the same but can be optimized independently

## Security Considerations

- **Network Security**: In Kubernetes, use NetworkPolicies to restrict access to the auth service
- **Secrets Management**: Use Kubernetes Secrets or Docker secrets for sensitive configuration
- **TLS**: Consider enabling TLS between Traefik and the auth service in production

## Getting Help

If you encounter issues during migration:

1. Check the [README.md](README.md) for detailed documentation
2. Review the [k8s/README.md](k8s/README.md) for Kubernetes-specific guidance
3. Open an issue on GitHub with your configuration and error logs
