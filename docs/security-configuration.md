# Security Configuration

Enduro exposes an operator dashboard and HTTP API, but it does not currently
implement application-level user authentication or authorization. Production
deployments must restrict access with an external access-control layer, such as
a reverse proxy, edge gateway, identity-aware proxy, VPN, network policy, or
service mesh.

## Baseline

Use one public origin for the dashboard and API whenever possible. This is the
recommended browser security posture:

```toml
[api]
allowedOrigins = []
```

With this setting, browser requests from the same origin are allowed and unsafe
cross-origin browser requests are rejected before they reach state-changing API
handlers. Typical command-line or service-to-service API clients are unaffected
because they do not send browser origin headers.

If `api.allowedOrigins` is omitted, Enduro uses open CORS mode:

```toml
[api]
allowedOrigins = ["*"]
```

Open CORS mode permits CORS responses for any origin and disables Enduro's
cross-origin write protection. Use it only when that behavior is intentional.

## Cross-Origin Access

Use explicit origins only when the dashboard and API must be served from
different origins:

```toml
[api]
allowedOrigins = ["https://dashboard.example.org"]
```

Origins must match the browser `Origin` header exactly, including scheme and
port. Avoid broad origins.

## Why This Applies To An API

CSRF-style protection is relevant when browsers can reach state-changing API
endpoints and credentials are attached automatically by the browser. This can
happen even when Enduro relies on an external access-control layer, for example
when that layer uses cookies, client certificates, or HTTP authentication.

CORS controls whether a browser lets a site read cross-origin responses. It is
not a write protection by itself. Enduro also rejects unsafe cross-origin
browser requests so a malicious site cannot trigger Enduro operations through a
user's browser, even if it cannot read the response.

## Content Security Policy

Enduro can send a configured `Content-Security-Policy` header together with
other browser hardening headers:

```toml
[api]
contentSecurityPolicy = "default-src 'self'; base-uri 'self'; object-src 'none'; frame-ancestors 'none'; connect-src 'self'; img-src 'self' data:; font-src 'self' data:; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'"
```

Adjust the policy if a deployment intentionally loads assets or connects to APIs
on other origins.

## Checklist

1. Keep Enduro bound to a private interface unless it is intentionally exposed by
   the external access-control layer.
2. Require authentication and authorization before traffic reaches Enduro.
3. Serve the dashboard and API from the same origin when possible.
4. Configure `api.allowedOrigins` explicitly for split-origin deployments.
5. Avoid `allowedOrigins = ["*"]` in deployments that rely on browser-attached
   credentials.
6. Enable a Content Security Policy after validating the deployed dashboard and
   any required external origins.
