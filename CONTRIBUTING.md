# Contributing to ztpanel

Thanks for considering a contribution. ztpanel is a small project and
this is meant to stay a short document.

## Setup

```sh
npm --prefix web install
npm --prefix web run build   # builds web/dist, which the Go binary embeds
go build ./cmd/ztpanel
```

You'll need a real or fake ZeroTier controller to run it against. For
a fake one, any HTTP server answering `/status`, `/controller/network`
etc. the way `zerotier-one`'s local API does will do -- the test
suite's `internal/api` package has one (`fakeController` in
`testutil_test.go`) if you want a reference.

## Running the tests

```sh
go test ./...                # backend
npm --prefix web run build   # frontend: type-checks and builds
```

`internal/api`'s tests run the full HTTP server against the in-memory
fake controller mentioned above, so nothing external is required.

If you touch `cmd/ztpanel/main.go`'s routing, `go test` alone won't
catch a mistake there: those tests exercise `internal/api.Server`
directly via `httptest.NewServer`, bypassing the outer mux entirely.
Build the real binary and curl it:

```sh
go build -o /tmp/ztpanel ./cmd/ztpanel
ZTPANEL_DATA_DIR=/tmp/ztpanel-data ZTPANEL_ZT_AUTHTOKEN=/path/to/any/file ./tmp/ztpanel &
curl -s localhost:8787/healthz
```

This is exactly how a routing bug slipped through once before (`/mcp`
silently fell through to the SPA catch-all) -- `go test` was green the
whole time.

## Code style

- No comments explaining *what* code does -- names should already say
  that. A comment earns its place only when it explains a non-obvious
  *why* (a constraint, an invariant, a workaround).
- Match the existing patterns: flat JSON-file stores behind a
  `sync.Mutex` for small, append-light state (see `internal/authsvc`,
  `internal/orgsvc`, `internal/auditsvc`, `internal/apitoken`) --
  there's no database, and that's deliberate.
- Keep handlers thin; put logic that's reused by both a REST handler
  and an MCP tool into a shared method on `Server` (see
  `personalNetworks`, `networkMembers`, `orgMembership` in
  `internal/api`).
- Go: run `go vet ./...` before sending a PR; there's no separate
  linter config.
- Vue/TypeScript: `npm run build` runs `vue-tsc` first, so type errors
  fail the build.

## Pull requests

- Keep them focused -- one change, one PR, with tests for whatever
  behavior changed.
- Update the README if you add or remove something listed under
  "Status" or change anything documented under "MCP / API access" or
  "Configuration".
- CI (`.github/workflows/ci.yml`) runs `go build`, `go vet`, `go test`
  and the frontend build on every PR; it needs to be green.

## Reporting security issues

Please don't open a public issue for a security vulnerability. Open a
[private security advisory](https://github.com/antonio-damaso/ztpanel/security/advisories/new)
on GitHub instead.
