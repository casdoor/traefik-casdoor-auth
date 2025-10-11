# casdoor-traefik-plugin

## Install

### Pre-built Docker Images

Official images are automatically built and published to GitHub Container Registry:
```bash
docker pull ghcr.io/casdoor/traefik-casdoor-auth:latest
```

Community-maintained image: https://github.com/lostb1t/traefik-casdoor-auth

## Deployment Options

This project supports two deployment modes:

1. **Traefik Plugin Mode** (Experimental): Uses Traefik's local plugin system
2. **Standalone ForwardAuth Mode** (Recommended for Production): Runs as a standalone service using Traefik's ForwardAuth middleware

### Standalone ForwardAuth Deployment

The standalone mode is recommended for Kubernetes and production deployments as it doesn't rely on Traefik's experimental plugin system.

#### Using Docker

You can use the pre-built image or build your own:

**Using pre-built image:**
```bash
docker run -d \
  -p 9999:9999 \
  -v $(pwd)/conf/plugin.json:/app/conf/plugin.json:ro \
  --name traefik-casdoor-auth \
  ghcr.io/casdoor/traefik-casdoor-auth:latest
```

**Building your own image:**
```bash
docker build -t traefik-casdoor-auth .
docker run -d \
  -p 9999:9999 \
  -v $(pwd)/conf/plugin.json:/app/conf/plugin.json:ro \
  --name traefik-casdoor-auth \
  traefik-casdoor-auth
```

#### Using Docker Compose

Use the provided `docker-compose.standalone.yml`:
```bash
docker-compose -f docker-compose.standalone.yml up -d
```

#### Kubernetes Deployment

Example Kubernetes deployment manifests are provided in the `k8s/` directory (see below for configuration).

#### Traefik Configuration for Standalone Mode

Configure Traefik to use ForwardAuth middleware instead of the plugin:

**Static Configuration (traefik.yml):**
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

**Dynamic Configuration (dynamic.yml):**
```yaml
http:
  routers:
    my-router:
      rule: Host(`your-domain.local`)
      service: my-service
      entryPoints:
        - web
      middlewares:
        - casdoor-auth

  services:
    my-service:
      loadBalancer:
        servers:
          - url: http://your-backend-service:8080
    
    # ForwardAuth service
    casdoor-auth-service:
      loadBalancer:
        servers:
          - url: http://traefik-casdoor-auth:9999

  middlewares:
    casdoor-auth:
      forwardAuth:
        address: http://traefik-casdoor-auth:9999/auth
        trustForwardHeader: true
```

## 1. Introduction

This is a solution for traefik which can be used to add authentication to any http service managed by traefik. This solution consists 2 parts: 

- A traefik plugin used to intercept the http request , forward to a special webhook(which is the second part of this plugin) and get instrcutions about what to do next from the webhook. 
- A webhook which analyze the http request forwarded from the traefik plugin, and give out further instructions to traefik plugin and possibly cache it.

## 2. Quick start

### 2.1 Prerequisite

You need to have traefik,docker and casdoor installed.<br>

casdoor:<https://casdoor.org/><br>
traefik: <https://doc.traefik.io/><br>

You also need to understand how traefik configurations works. We use yml configs here to exemplify. In case that you are not using the same way to configurate traefik, you need to convert the configurations into correct format you need by yourself.<br>

The webhook itself is an app of casdoor(What's this? see <https://casdoor.org/docs/basic/core-concepts>). Register this application in casdoor and get the client id and client secret,casdoorOrganization name and casdoorApplication name.(If you don't know how to do this, see <https://casdoor.org/docs/application/config/>)

### 2.2 modify the configuration

### 2.2.1 modify static configuration for traefik

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

Here, we specify that we are using a local plugin (instead of an online plugin) named 'example'. The model name must be exactly the path name relative to the 'plugins-local/src' folder in the workspace. You can see that there is indeed codes of plugins in plugins-local/src/github.com/casdoor/plugindemo.In addition, this name is also the same with the name declared in the plugin(plugins-local/src/github.com/casdoor/plugindemo.traefik.yml) If you want to change the path, make sure you change them all.<br>
We also point out that the dynamic configuration file is dev.yml.

### 2.2.2 dynamic configuration file

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

`http.routers.myroute` specified we want to apply a middleware called 'my-plugin' to service'webhook.domain.local'. `middlewares`paragraph specify that this plugin is a 'example'plugin(we defined in static configuration), and give out a parameter 'multationWebhook', which is the endpoint of the webhook. If you want to use a url other than this, you should change it here.

### 2.2.3 webhook configuration file (conf/plugin.json)

```json
{
    "casdoorEndpoint":"http://webhook.domain.local:8000", 
    "casdoorClientId":"88b2457a123984b48392",
    "casdoorClientSecret":"1a3f5eb7990b92f135a78fab5d0327890f2ae8df",
    "casdoorOrganization":"Traefik ForwardAuth",
    "casdoorApplication":"TraefikForwardAuthPlugin",
    "pluginEndPoint":"http://webhook.domain.local:9999"
}
```

- "casdoorEndpoint": endpoint of casdoor
- "casdoorClientId": casdoor client id
- "casdoorClientSecret": casdoor client secret
- "casdoorOrganization":organization name which casdoor app belongs to
- "casdoorApplication": casdoor app name
- "pluginEndPoint": the url of this webhook.

### 2.2.4 Run

#### modify host

modify host files of your instance to point 'webhook.domain.local' to localhost

#### start a example service

```
docker compose up -d
```

this command runs a 'who am i' container at port 5000, which is the official example service used by traefik. I am quite sure that you should be familiar with this if you have ever tried traefik. This container start a web service, which always return information about your http request without any other authentication.

#### start the traefik

```
sudo traefik --configFile="traefik.yml" --log.level=DEBUG
```

### start the webhook 

```shell
go run cmd/webhook/main.go  -configFile="conf/plugin.json"
```

Visit: http://webhook.domain.local. If you have nevered logged in, you will be redirected to the casdoor login page. If you have logged in through casdoor before, you will see the 'whoami'output: the reflection of your http request.  

## 3. How it works?

The traefik plugin will intercept any request for the protected service and forward this request to our webhook. After the webhook responsed , if the response status code is 2xx then, the original request will be modified based on the instruction given by the webhook (in the response body) and allowed to proceed. If the reponse status is not 2xx then the request will not proceed and the resoponse body as well as the status code given out by out webhook will be returned without any modification

Once out webhook received the request forwarede by out plugin, it will check whether there exists a special cookie set by our webhook. If the special cookie doesn't exist, the webhook will return a 302 redirect, redirecting the user to casdoor login page with a proper redirect url pointing to another redirect handler of our webhook. Besides, the original request will be recorded.

After the user logged in, the user will be redirected to the redirect handler mentioned above. This time we will first trying to require the OAuthToken to check whethre the client code is legit set up the cookie, and redirect the user to the original URL he wanted to visit.


If the user is redirected to the original URL he wanted to visit, this request will be forwarded to our webhook again. This time after confirming the existence of cookie, we will instruct the plugin to alter the requset to be the same with the first original request (because we have recorded it.)
Thus without making the service be aware of the existence of authentication procedure, the user is authentication and the service is properly protected.

