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
| Default admin password | Hardcoded (with a known bug)  | Set during setup                | Random, generated on first run  |
| Mobile-friendly UI     | No                              | Partial                          | Yes (mobile-first)              |
| Docker image size*     | ~106 MiB (bundles its own `zerotier-one`) | ~251 MiB app + ~90 MiB Postgres + ~6 MiB zerotier sidecar | ~12 MiB binary, no extra containers |
| Idle memory (RAM)*     | ~85 MiB (whole container)       | not benchmarked here             | ~10 MiB RSS, single process     |

ztnet has the most mature feature set and a larger community; ztpanel
trades some of that maturity for a lighter deployment footprint (no
external database) and a from-scratch, less cluttered UI.

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

- Login, sessions (stateless signed cookies), per-user password change
- First-run bootstrap: a random admin password is generated and printed
  once -- there is no hardcoded default password
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

On first run, ztpanel creates an `admin` user with a random password
and prints it once to stdout -- save it, you'll be asked to change it
on first login.

### Configuration (environment variables)

| Variable                | Default                                  | Meaning                                  |
|--------------------------|-------------------------------------------|-------------------------------------------|
| `ZTPANEL_LISTEN`         | `:8787`                                   | HTTP listen address                       |
| `ZTPANEL_ZT_URL`         | `http://127.0.0.1:9993`                   | ZeroTier controller API base URL          |
| `ZTPANEL_ZT_AUTHTOKEN`   | `/var/lib/zerotier-one/authtoken.secret`  | Path to the controller's auth token file  |
| `ZTPANEL_DATA_DIR`       | `./data`                                  | Where ztpanel stores its own users/session key |

## Testing

```sh
go test ./...                       # backend
npm --prefix web run build          # frontend (type-checks + builds)
```

The `internal/api` tests run the full HTTP server against an in-memory
fake of the ZeroTier controller, so they don't need a real one running.

## License

MIT, see [LICENSE](LICENSE).
