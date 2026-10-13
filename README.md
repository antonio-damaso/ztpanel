# ztpanel

A self-hosted web UI for a [ZeroTier](https://www.zerotier.com/) network
controller: a single Go binary (REST API + embedded Vue 3 / PrimeVue
frontend) talking to the controller's local JSON API.

Built as a from-scratch alternative to
[ztncui](https://github.com/key-networks/ztncui), aiming for a cleaner,
less cluttered UI, and to
[`ztnet`](https://github.com/sinamics/ztnet) (Next.js/Prisma) when a
single-binary deployment with no external database is preferred.

## Comparison

|                       | ztncui                        | ztnet                          | ztpanel                        |
|-----------------------|--------------------------------|---------------------------------|---------------------------------|
| Stack                 | Express + Pug + jQuery         | Next.js + Prisma                | Go + Vue 3 / PrimeVue           |
| Deployment            | Single binary/container        | Needs a database (Postgres)     | Single binary, no database      |
| ZeroTier daemon       | Internal (bundled in the same container) | External (separate sidecar container) | External (connects to a controller you already run) |
| License               | GPLv3                          | GPLv3                           | MIT                             |
| Organizations         | No                              | Yes                              | Yes                              |
| Activity log          | No                              | Yes                              | Yes                              |
| Auth token editing     | No (documented as missing)     | Yes                              | Yes                              |
| Default admin password | Hardcoded (with a known bug)  | Set during setup                | Chosen on a first-run setup page |
| Mobile-friendly UI     | No                              | Partial                          | Yes (mobile-first)              |
| REST API for scripts   | No (session-cookie-only internal routes, not a documented API) | Yes -- `/api/v1/*`, custom `x-ztnet-auth` header | Yes -- `/api/*`, `Authorization: Bearer` token |
| LLM / MCP integration  | No                              | No                                | Yes -- `/mcp` endpoint, same Bearer token as the REST API |
| Docker image size*     | ~106 MiB (bundles its own `zerotier-one`) | ~251 MiB app + ~90 MiB Postgres + ~6 MiB zerotier sidecar | ~12 MiB binary, no extra containers |
| Idle memory (RAM)*     | ~85 MiB (whole container)       | not benchmarked here             | ~10 MiB RSS, single process     |

ztnet has the most mature feature set and a larger community; ztpanel
trades some of that maturity for a lighter deployment footprint (no
external database) and a from-scratch, less cluttered UI.

Neither ztncui's nor ztnet's READMEs document an API, so the "REST
API for scripts" row was checked by reading their source instead:
ztncui's `/network/*` routes (`src/routes/zt_controller.js`) are
internal AJAX endpoints for its own Pug views, guarded by
`req.session.user` only (`src/controllers/auth.js`) -- there's no
token path in. ztnet does have a real one, versioned and tested
(`src/pages/api/v1/**`), authenticated via a custom `x-ztnet-auth`
header (`src/utils/apiRouteAuth.ts`) and covering networks, members,
organizations, users and stats.

\* Docker image sizes are the published compressed sizes from Docker
Hub (`keynetworks/ztncui:latest`, `sinamics/ztnet:latest`,
`postgres:15.2-alpine`, `zyclonite/zerotier:1.14.2`), checked when this
table was written -- they'll drift over time. The ztpanel figure is
its idle RSS measured directly on a local build. The ztncui figure is
`docker stats` on a real, running instance (its own Node.js process
inside the container accounts for ~80 MiB of that, with `zerotier-one`
and supervisord making up the rest). ztnet was not run in this
environment, so no RAM number is given for it -- expect it to use
noticeably more than ztncui, since it runs a full Next.js server
alongside a separate Postgres server.

## Status

Early / work in progress. Currently implemented:

- Login, sessions (stateless signed cookies, `Secure` behind a TLS
  reverse proxy), per-user password change, per-IP rate limiting on
  `/api/login` and `/api/setup`, constant-time password checks
  (no username-enumeration timing side channel)
- First-run setup page: choose the admin username/password yourself in
  the browser (and see whether the ZeroTier controller is reachable)
  instead of digging a generated password out of the container logs --
  there is no hardcoded default password either way
- Networks: list, create, rename, delete, private/public toggle, MTU,
  multicast limit
- Members: list with live online/relay/offline status and latency,
  rename, authorize, active-bridge toggle, IP assignment, delete
- Network settings: routes, IP assignment pools, IPv4/IPv6 assign
  modes, DNS, auth tokens (the last of which the original ztncui never
  implemented editing for)
- An "Advanced" panel with the network's raw controller JSON, collapsed
  by default, for debugging
- Panel user management (create, delete, reset password)
- Organizations: group networks and scope who can see/manage them
  (admin/user roles per organization), the way ztnet does -- ztpanel
  keeps this entirely in its own store, since the ZeroTier controller
  itself has no concept of organizations
- Activity log: who did what, when, across networks/members/users/orgs
- Mobile-first responsive UI (off-canvas sidebar below 768px)
- API tokens and an MCP endpoint, so an LLM (Claude or otherwise) can
  manage networks/members directly -- see [MCP / API access](#mcp--api-access);
  tokens can be scoped read-only and/or set to expire
- An unauthenticated `/healthz` liveness check for container
  orchestration and reverse proxies
- Automated tests for every backend package and the full API surface
  (`go test ./...`), run against a fake ZeroTier controller

## Architecture

- `cmd/ztpanel` -- entrypoint
- `internal/ztapi` -- thin client for the ZeroTier controller's local
  JSON API (`127.0.0.1:9993`, authenticated via `authtoken.secret`)
- `internal/authsvc` -- ztpanel's own users (who may use the panel) and
  sessions; independent from ZeroTier's own identities
- `internal/orgsvc` -- organizations: grouping networks and scoping
  access by role, stored locally since the controller has no such
  concept
- `internal/auditsvc` -- the append-only activity log
- `internal/api` -- REST handlers
- `internal/config` -- environment-based configuration
- `web/` -- Vue 3 + PrimeVue frontend (Vite), embedded into the Go
  binary at build time via `go:embed`

## Running it

Requires a `zerotier-one` controller already running on the host (or
reachable over the network) -- ztpanel does not run its own ZeroTier
daemon.

### Docker

A prebuilt image is published to GHCR on every push to `main` and on
every `v*` tag:

```sh
docker run -d \
  -p 8787:8787 \
  -v /var/lib/zerotier-one/authtoken.secret:/var/lib/zerotier-one/authtoken.secret:ro \
  -v ztpanel-data:/data \
  --network host \
  ghcr.io/antonio-damaso/ztpanel:latest
```

`--network host` is the simplest way to let the container reach the
controller at `127.0.0.1:9993`; alternatively set `ZTPANEL_ZT_URL` to
wherever the controller is actually reachable from the container.

### Prebuilt binary

Each [release](https://github.com/antonio-damaso/ztpanel/releases/latest)
ships a single, statically-linked `ztpanel-linux-amd64` binary (the
Vue frontend is embedded in it) alongside the `Dockerfile`:

```sh
curl -LO https://github.com/antonio-damaso/ztpanel/releases/latest/download/ztpanel-linux-amd64
chmod +x ztpanel-linux-amd64

ZTPANEL_ZT_AUTHTOKEN=/var/lib/zerotier-one/authtoken.secret \
ZTPANEL_DATA_DIR=./data \
./ztpanel-linux-amd64
```

### From source

```sh
# 1. Build the frontend
npm --prefix web install
npm --prefix web run build

# 2. Build the Go binary (the Vue build above is embedded into it)
go build -o ztpanel ./cmd/ztpanel

# 3. Run it
ZTPANEL_ZT_AUTHTOKEN=/var/lib/zerotier-one/authtoken.secret \
ZTPANEL_DATA_DIR=./data \
./ztpanel
```

Open it in a browser. On first run, with no users yet, ztpanel serves
a setup page instead of the login screen -- pick the admin username
and password there (the page also shows whether it could reach the
ZeroTier controller) and you're logged straight in.

### Configuration (environment variables)

| Variable                | Default                                  | Meaning                                  |
|--------------------------|-------------------------------------------|-------------------------------------------|
| `ZTPANEL_LISTEN`         | `:8787`                                   | HTTP listen address                       |
| `ZTPANEL_ZT_URL`         | `http://127.0.0.1:9993`                   | ZeroTier controller API base URL          |
| `ZTPANEL_ZT_AUTHTOKEN`   | `/var/lib/zerotier-one/authtoken.secret`  | Path to the controller's auth token file  |
| `ZTPANEL_DATA_DIR`       | `./data`                                  | Where ztpanel stores its own users/session key |

ztpanel itself speaks plain HTTP only -- put a TLS-terminating reverse
proxy (Caddy, nginx, Traefik...) in front of it for any deployment
reachable outside `localhost`. The session cookie is marked `Secure`
whenever the proxy forwards `X-Forwarded-Proto: https`, so make sure
your proxy sets that header.

## MCP / API access

ztpanel exposes a [Model Context Protocol](https://modelcontextprotocol.io/)
endpoint at `POST /mcp`, so an LLM can list/create/update/delete
networks and members directly. Organizations and the activity log are
exposed read-only; membership and org structure stay a web-UI-only
action.

Authentication is a long-lived Bearer API token, not OAuth: ztpanel is
a single, self-hosted binary with no authorization-server
infrastructure of its own, so a static token -- generated once in
**Account -> API tokens** -- is the simpler fit.

There is only one kind of token, and it authenticates both paths: the
`/api/*` REST API used by the SPA and by scripts, and the `/mcp`
endpoint used by MCP clients. The same `Authorization: Bearer <token>`
header works on either one -- it's two protocols over one
credential, not two separate auth systems:

```sh
# Plain REST, e.g. from a script -- no MCP involved.
curl https://your-ztpanel-host/api/networks \
  -H "Authorization: Bearer ztpanel_xxxxx"

# Same token, MCP's JSON-RPC envelope instead of REST.
curl -X POST https://your-ztpanel-host/mcp \
  -H "Authorization: Bearer ztpanel_xxxxx" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_networks"}}'
```

With [Claude Code](https://claude.com/claude-code):

```sh
claude mcp add --transport http ztpanel https://your-ztpanel-host/mcp \
  --header "Authorization: Bearer ztpanel_xxxxx"
```

Note this does **not** work with Claude.ai/Claude Desktop's
"Connectors" UI, which requires a remote MCP server to be an OAuth
authorization server -- ztpanel isn't one. It works with any MCP
client that can send a static Bearer header (Claude Code, most
self-hosted/CLI clients).

### Token scope and expiry

A token can optionally be **read-only** (list/get only -- no creates,
updates or deletes, on either `/api/*` or `/mcp`) and/or set to
**expire** after a number of days. Both are chosen when the token is
created, in **Account -> API tokens**, and can't be changed
afterwards -- revoke it and create a new one instead. A read-only
token is the safer default for handing to an LLM that will mostly be
asked to look things up.

## Health check

`GET /healthz` is an unauthenticated liveness check (`{"status":"ok"}`
once the process is up) for container orchestration or a reverse
proxy -- it does not depend on the ZeroTier controller being
reachable. The Docker image's own `HEALTHCHECK` uses it. Controller
reachability itself is reported by the authenticated `GET /api/status`
(and surfaced in the UI), since there's no useful way to check it
without a logged-in session.

## Testing

```sh
go test ./...                       # backend
npm --prefix web run build          # frontend (type-checks + builds)
```

The `internal/api` tests run the full HTTP server against an in-memory
fake of the ZeroTier controller, so they don't need a real one running.
See [CONTRIBUTING.md](CONTRIBUTING.md) for more, including why that
alone isn't enough to catch every kind of bug here.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT, see [LICENSE](LICENSE).
