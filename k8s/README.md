# Kubernetes Deployment Guide

This directory contains Kubernetes manifests for deploying traefik-casdoor-auth as a standalone ForwardAuth service.

## Prerequisites

- Kubernetes cluster (1.19+)
- Traefik installed in your cluster
- Docker image built and available in your registry

## Quick Start

1. **Build and push the Docker image:**

```bash
# Build the image
docker build -t your-registry/traefik-casdoor-auth:latest .

# Push to your registry
docker push your-registry/traefik-casdoor-auth:latest
```

2. **Update the configuration:**

Edit `deployment.yaml` and update the ConfigMap with your Casdoor credentials:
- `casdoorEndpoint`: Your Casdoor server URL
- `casdoorClientId`: Your application's client ID
- `casdoorClientSecret`: Your application's client secret
- `casdoorOrganization`: Your Casdoor organization name
- `casdoorApplication`: Your Casdoor application name
- `pluginEndPoint`: The public URL where this service will be accessible

3. **Deploy to Kubernetes:**

```bash
kubectl apply -f deployment.yaml
```

4. **Configure Traefik ForwardAuth:**

Choose one of the following options:

### Option A: Using Traefik CRDs (IngressRoute)

Apply the IngressRoute example:

```bash
kubectl apply -f ingressroute-example.yaml
```

### Option B: Using Standard Ingress with Annotations

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-protected-app
  annotations:
    traefik.ingress.kubernetes.io/router.middlewares: default-casdoor-forwardauth@kubernetescrd
spec:
  rules:
  - host: your-app.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: your-backend-service
            port:
              number: 8080
```

## Configuration Options

### Environment Variables

The service can be configured using command-line flags or by mounting a configuration file.

Default configuration file path: `/app/conf/plugin.json`

### Scaling

The deployment is configured with 2 replicas by default. You can scale it up or down:

```bash
kubectl scale deployment traefik-casdoor-auth -n traefik-casdoor-auth --replicas=3
```

### Health Checks

The service exposes a health check endpoint at `/echo` which is used for liveness and readiness probes.

## Troubleshooting

Check the logs:
```bash
kubectl logs -n traefik-casdoor-auth -l app=traefik-casdoor-auth -f
```

Verify the service is running:
```bash
kubectl get pods -n traefik-casdoor-auth
kubectl get svc -n traefik-casdoor-auth
```

Test the auth endpoint:
```bash
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl -v http://traefik-casdoor-auth.traefik-casdoor-auth.svc.cluster.local:9999/echo
```
