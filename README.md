# ztpanel

A self-hosted web UI for a [ZeroTier](https://www.zerotier.com/) network
controller: a single Go binary (REST API + embedded Vue 3 / PrimeVue
frontend) talking to the controller's local JSON API.

Built as a from-scratch alternative to
[ztncui](https://github.com/key-networks/ztncui), aiming for a cleaner,
less cluttered UI, and to
[`ztnet`](https://github.com/tinola/ztnet) (Next.js/Prisma) when a
single-binary deployment with no external database is preferred.

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
- Basic panel user management (create, delete, reset password)

Planned: multi-user permissions and multi-organization support (scoping
networks/users to an organization, the way ztnet does).

## Architecture

- `cmd/ztpanel` -- entrypoint
- `internal/ztapi` -- thin client for the ZeroTier controller's local
  JSON API (`127.0.0.1:9993`, authenticated via `authtoken.secret`)
- `internal/authsvc` -- ztpanel's own users (who may use the panel) and
  sessions; independent from ZeroTier's own identities
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

## License

MIT, see [LICENSE](LICENSE).
