# Casdoor Forward Auth

<p align="center">
  <a href="#badge">
    <img alt="semantic-release" src="https://img.shields.io/badge/%20%20%F0%9F%93%A6%F0%9F%9A%80-semantic--release-e10079.svg">
  </a>
  <a href="https://github.com/casdoor/casdoor-forward-auth/actions/workflows/ci.yml">
    <img alt="GitHub Workflow Status (branch)" src="https://img.shields.io/github/actions/workflow/status/casdoor/casdoor-forward-auth/ci.yml?branch=master">
  </a>
  <a href="https://github.com/casdoor/casdoor-forward-auth/releases/latest">
    <img alt="GitHub Release" src="https://img.shields.io/github/v/release/casdoor/casdoor-forward-auth.svg">
  </a>
</p>

<p align="center">
  <a href="https://goreportcard.com/report/github.com/casdoor/casdoor-forward-auth">
    <img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/casdoor/casdoor-forward-auth?style=flat-square">
  </a>
  <a href="https://github.com/casdoor/casdoor-forward-auth/blob/master/LICENSE">
    <img src="https://img.shields.io/github/license/casdoor/casdoor-forward-auth?style=flat-square" alt="license">
  </a>
  <a href="https://github.com/casdoor/casdoor-forward-auth/issues">
    <img alt="GitHub issues" src="https://img.shields.io/github/issues/casdoor/casdoor-forward-auth?style=flat-square">
  </a>
  <a href="#">
    <img alt="GitHub stars" src="https://img.shields.io/github/stars/casdoor/casdoor-forward-auth?style=flat-square">
  </a>
  <a href="https://github.com/casdoor/casdoor-forward-auth/network">
    <img alt="GitHub forks" src="https://img.shields.io/github/forks/casdoor/casdoor-forward-auth?style=flat-square">
  </a>
  <a href="https://discord.gg/5rPsrAzK7S">
    <img alt="Casdoor" src="https://img.shields.io/discord/1022748306096537660?style=flat-square&logo=discord&label=discord&color=5865F2">
  </a>
</p>

Casdoor Forward Auth puts [Casdoor](https://casdoor.ai/) single sign-on in front of any web application, without changing the application. It works with the forward auth feature of reverse proxies:

- [Traefik](#traefik): `forwardAuth` middleware
- [Caddy](#caddy): `forward_auth` directive
- [Nginx](#nginx): `auth_request` module
- [Kibana and other apps](#kibana) behind any of the above

Signed-in users reach the application with their identity in request headers (`X-Forwarded-User`, `X-Forwarded-Email`, ...), everyone else is sent to the Casdoor login page first.

> This project was called `traefik-casdoor-auth` and needed a Traefik plugin. Since v2 it's a standalone service that works with Traefik's built-in `forwardAuth` middleware, so the plugin is gone. See [Upgrading from traefik-casdoor-auth](#upgrading-from-traefik-casdoor-auth).

## How it works

```
Browser ──► reverse proxy ──(every request)──► casdoor-forward-auth /auth
                 │                                   │
                 │   200 + X-Forwarded-User, ...  ◄──┤  valid session cookie
                 ▼                                   │
           your application                          └─ no session: 302 to /login ──► Casdoor login
                                                                                          │
           session cookie set, 302 back to the page ◄── /callback ◄── authorization code ◄┘
```

1. The reverse proxy asks `/auth` about every request. With a valid session cookie the answer is `200` with the identity headers, which the proxy copies into the request to your application.
2. Without a session, a page load is redirected to `/login`, which redirects to Casdoor with a random `state` kept in a signed, short-lived cookie. Other requests (`POST`, `PUT`, ...) get `401`, since they can't follow a login redirect.
3. Casdoor sends the user back to `/callback`. The service checks the `state`, exchanges the authorization code for an access token once, verifies the token's signature and audience, and stores the user in a signed `HttpOnly` session cookie.
4. The user is redirected back to the page they asked for. From now on every request is answered from the cookie, without calling Casdoor.

The service keeps no state on the server, so you can run several replicas behind a load balancer as long as they share the same `cookieSecret`.

## Quick start

### 1. Create an application in Casdoor

In Casdoor, add an application (or use an existing one) and note its **Client ID** and **Client secret**. Add the callback of this service to its **Redirect URLs**:

```
https://auth.example.com/callback
```

where `https://auth.example.com` is the public URL of casdoor-forward-auth (`externalUrl` below).

### 2. Run casdoor-forward-auth

With Docker:

```bash
docker run -d -p 9999:9999 \
  -e CASDOOR_ENDPOINT=https://door.casdoor.com \
  -e CLIENT_ID=<client ID> \
  -e CLIENT_SECRET=<client secret> \
  -e EXTERNAL_URL=https://auth.example.com \
  -e COOKIE_DOMAIN=example.com \
  -e COOKIE_SECRET=$(openssl rand -hex 32) \
  ghcr.io/casdoor/casdoor-forward-auth:latest
```

Or from source (Go 1.23+):

```bash
go install github.com/casdoor/casdoor-forward-auth@latest
casdoor-forward-auth -config config.json
```

Generate `COOKIE_SECRET` once and keep it: changing it signs everybody out.

### 3. Configure your reverse proxy

See [Traefik](#traefik), [Caddy](#caddy) or [Nginx](#nginx) below.

## Configuration

Settings come from a JSON file (`-config config.json` or the `CONFIG_FILE` environment variable, see [conf/config.json](conf/config.json)) and/or environment variables. Environment variables override the file.

| JSON key | Environment variable | Default | Description |
|---|---|---|---|
| `casdoorEndpoint` | `CASDOOR_ENDPOINT` | required | URL of the Casdoor server, e.g., `https://door.casdoor.com` |
| `clientId` | `CLIENT_ID` | required | Client ID of the Casdoor application |
| `clientSecret` | `CLIENT_SECRET` | required | Client secret of the Casdoor application |
| `externalUrl` | `EXTERNAL_URL` | required | Public URL of this service as the browser sees it, e.g., `https://auth.example.com`. It may have a path, e.g., `https://app.example.com/_auth`, then all endpoints live under that path. `<externalUrl>/callback` must be a Redirect URL of the Casdoor application |
| `cookieSecret` | `COOKIE_SECRET` | required | Secret of at least 32 characters for signing the cookies |
| `cookieDomain` | `COOKIE_DOMAIN` | empty | Domain of the session cookie, e.g., `example.com` to share the session with all subdomains. Required when the applications aren't on the host of `externalUrl`. Must contain the host of `externalUrl` |
| `cookieName` | `COOKIE_NAME` | `casdoor_forward_auth` | Name of the session cookie |
| `sessionTtl` | `SESSION_TTL` | `24h` | Session lifetime, a Go duration like `8h` or `30m`. Never longer than the access token issued by Casdoor |
| `allowedRedirectDomains` | `ALLOWED_REDIRECT_DOMAINS` (comma separated) | host of `externalUrl` and `.<cookieDomain>` | Where the user may be sent back after login or logout. `example.com` allows that host only, `.example.com` allows it and all subdomains. Anything else goes to `externalUrl` instead, so the login can't be used as an open redirect |
| `certificate` | `CERTIFICATE` | empty | PEM certificate for verifying access tokens. When empty, the certificate is looked up in Casdoor's JWKS (`/.well-known/jwks`) by the token's key ID, which also follows certificate changes |
| `listenAddr` | `LISTEN_ADDR` | `:9999` | Address to listen on |

## Endpoints

| Endpoint | Used by | Behavior |
|---|---|---|
| `/auth` | Traefik, Caddy | `200` + identity headers when signed in; otherwise `302` to the login for `GET`/`HEAD` and `401` for other methods (from `X-Forwarded-Method`) |
| `/verify` | Nginx | `200` + identity headers when signed in, otherwise `401` |
| `/login?rd=<url>` | browser | Starts the login and returns to `rd` afterwards |
| `/callback` | Casdoor | OAuth callback |
| `/logout?rd=<url>` | browser | Clears the session cookie, then redirects to `rd` (if given) |
| `/healthz` | monitoring | Returns `ok` |
| `/` | browser | Shows who is signed in |

The identity headers are:

| Header | Value |
|---|---|
| `X-Forwarded-User` | User name, e.g., `alice` |
| `X-Forwarded-User-Id` | User ID |
| `X-Forwarded-Organization` | Organization of the user, e.g., `built-in` |
| `X-Forwarded-Email` | Email address |
| `X-Forwarded-Groups` | Comma-separated groups, e.g., `built-in/dev,built-in/ops` |
| `X-Forwarded-Roles` | Comma-separated role names |

All of them are always present (possibly empty), so the reverse proxy replaces whatever the client sent in the same headers. Make sure your application is only reachable through the reverse proxy, otherwise anyone can send these headers directly.

## Traefik

Docker labels (a complete example is in [examples/traefik/docker-compose.yml](examples/traefik/docker-compose.yml)):

```yaml
services:
  casdoor-forward-auth:
    image: ghcr.io/casdoor/casdoor-forward-auth:latest
    environment:
      CASDOOR_ENDPOINT: https://door.casdoor.com
      CLIENT_ID: <client ID>
      CLIENT_SECRET: <client secret>
      EXTERNAL_URL: https://auth.example.com
      COOKIE_DOMAIN: example.com
      COOKIE_SECRET: <random string of at least 32 characters>
    labels:
      - traefik.enable=true
      - traefik.http.routers.casdoor-auth.rule=Host(`auth.example.com`)
      - traefik.http.services.casdoor-auth.loadbalancer.server.port=9999
      - traefik.http.middlewares.casdoor.forwardauth.address=http://casdoor-forward-auth:9999/auth
      - traefik.http.middlewares.casdoor.forwardauth.authResponseHeaders=X-Forwarded-User,X-Forwarded-User-Id,X-Forwarded-Organization,X-Forwarded-Email,X-Forwarded-Groups,X-Forwarded-Roles

  whoami:
    image: traefik/whoami
    labels:
      - traefik.enable=true
      - traefik.http.routers.whoami.rule=Host(`app.example.com`)
      - traefik.http.routers.whoami.middlewares=casdoor
```

The same with the file provider:

```yaml
http:
  middlewares:
    casdoor:
      forwardAuth:
        address: http://casdoor-forward-auth:9999/auth
        authResponseHeaders:
          - X-Forwarded-User
          - X-Forwarded-User-Id
          - X-Forwarded-Organization
          - X-Forwarded-Email
          - X-Forwarded-Groups
          - X-Forwarded-Roles

  routers:
    casdoor-auth:
      rule: Host(`auth.example.com`)
      service: casdoor-auth
    app:
      rule: Host(`app.example.com`)
      service: app
      middlewares:
        - casdoor

  services:
    casdoor-auth:
      loadBalancer:
        servers:
          - url: http://casdoor-forward-auth:9999
    app:
      loadBalancer:
        servers:
          - url: http://app:8080
```

Don't put the `casdoor` middleware on the router of casdoor-forward-auth itself.

### Without a separate host

If you only have one host, mount the service under a path of the application, e.g., `EXTERNAL_URL=https://app.example.com/_auth` (no `COOKIE_DOMAIN` needed), and route that path to it without the middleware:

```yaml
  routers:
    casdoor-auth:
      rule: Host(`app.example.com`) && PathPrefix(`/_auth`)
      service: casdoor-auth
    app:
      rule: Host(`app.example.com`)
      service: app
      middlewares:
        - casdoor
```

with `forwardAuth.address: http://casdoor-forward-auth:9999/_auth/auth`, and `https://app.example.com/_auth/callback` as the Redirect URL in Casdoor.

## Caddy

```
auth.example.com {
	reverse_proxy casdoor-forward-auth:9999
}

app.example.com {
	forward_auth casdoor-forward-auth:9999 {
		uri /auth
		copy_headers X-Forwarded-User X-Forwarded-User-Id X-Forwarded-Organization X-Forwarded-Email X-Forwarded-Groups X-Forwarded-Roles
	}
	reverse_proxy app:8080
}
```

## Nginx

Nginx's `auth_request` only understands `2xx`, `401` and `403`, so it calls `/verify` and redirects to the login itself on `401`:

```nginx
server {
    server_name auth.example.com;

    location / {
        proxy_pass http://casdoor-forward-auth:9999;
        proxy_set_header Host $host;
    }
}

server {
    server_name app.example.com;

    location = /_casdoor_verify {
        internal;
        proxy_pass http://casdoor-forward-auth:9999/verify;
        proxy_pass_request_body off;
        proxy_set_header Content-Length "";
    }

    location @casdoor_login {
        return 302 https://auth.example.com/login?rd=$scheme://$http_host$request_uri;
    }

    location / {
        auth_request /_casdoor_verify;
        error_page 401 = @casdoor_login;

        auth_request_set $casdoor_user $upstream_http_x_forwarded_user;
        auth_request_set $casdoor_user_id $upstream_http_x_forwarded_user_id;
        auth_request_set $casdoor_organization $upstream_http_x_forwarded_organization;
        auth_request_set $casdoor_email $upstream_http_x_forwarded_email;
        auth_request_set $casdoor_groups $upstream_http_x_forwarded_groups;
        auth_request_set $casdoor_roles $upstream_http_x_forwarded_roles;
        proxy_set_header X-Forwarded-User $casdoor_user;
        proxy_set_header X-Forwarded-User-Id $casdoor_user_id;
        proxy_set_header X-Forwarded-Organization $casdoor_organization;
        proxy_set_header X-Forwarded-Email $casdoor_email;
        proxy_set_header X-Forwarded-Groups $casdoor_groups;
        proxy_set_header X-Forwarded-Roles $casdoor_roles;

        proxy_pass http://app:8080;
    }
}
```

## Kibana

To put Kibana (or Grafana, Prometheus, an internal admin page, ...) behind Casdoor, protect its host with any of the setups above, e.g., with Nginx replace `proxy_pass http://app:8080` by `proxy_pass http://kibana:5601`. Only expose Kibana through the reverse proxy. This replaces [elk-auth-casdoor](https://github.com/casdoor/elk-auth-casdoor).

Applications that can trust a header for the user name, such as Grafana's [auth proxy](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-authentication/auth-proxy/) (`header_name = X-Forwarded-User`), can also sign the user in automatically.

## Logout

`/logout` ends the session of casdoor-forward-auth only. The user stays signed in to Casdoor, so the next visit to a protected page goes through Casdoor without asking for the password again (unless the Casdoor session has ended too). `rd` must be inside `allowedRedirectDomains`.

## Upgrading from traefik-casdoor-auth

- The Traefik plugin (`plugins-local`, `experimental.localPlugins`) is no longer needed: remove it and use the built-in `forwardAuth` middleware as shown in [Traefik](#traefik).
- The config file changed: `casdoorClientId` → `clientId`, `casdoorClientSecret` → `clientSecret`, `pluginEndpoint` → `externalUrl`; `casdoorOrganization` and `casdoorApplication` are no longer needed; `cookieSecret` is new and required. The flag is now `-config` instead of `-configFile`.
- The Redirect URL in Casdoor stays `<externalUrl>/callback`.
- The old version exchanged the authorization code again on every request, which fails after the first one, and replayed the request body after the login. Now a signed session cookie is used, and only page loads are redirected to the login.

## Development

```bash
go test ./...
go run . -config conf/config.json
```

## License

[Apache-2.0](LICENSE)
