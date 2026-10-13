---
sidebar_position: 3
---

# Configuration

## Environment variables

| Variable                | Default                                  | Meaning                                  |
|--------------------------|-------------------------------------------|-------------------------------------------|
| `ZTPANEL_LISTEN`         | `:8787`                                   | HTTP listen address                       |
| `ZTPANEL_ZT_URL`         | `http://127.0.0.1:9993`                   | ZeroTier controller API base URL          |
| `ZTPANEL_ZT_AUTHTOKEN`   | `/var/lib/zerotier-one/authtoken.secret`  | Path to the controller's auth token file  |
| `ZTPANEL_DATA_DIR`       | `./data`                                  | Where ztpanel stores its own users/session key |

ztpanel itself speaks plain HTTP only — put a TLS-terminating reverse
proxy (Caddy, nginx, Traefik...) in front of it for any deployment
reachable outside `localhost`. The session cookie is marked `Secure`
whenever the proxy forwards `X-Forwarded-Proto: https`, so make sure
your proxy sets that header.

## Health check

`GET /healthz` is an unauthenticated liveness check (`{"status":"ok"}`
once the process is up) for container orchestration or a reverse
proxy — it does not depend on the ZeroTier controller being
reachable. The Docker image's own `HEALTHCHECK` uses it. Controller
reachability itself is reported by the authenticated `GET /api/status`
(and surfaced in the UI), since there's no useful way to check it
without a logged-in session.
