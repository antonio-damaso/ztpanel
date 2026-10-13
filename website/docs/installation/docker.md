---
sidebar_position: 1
---

# Docker

ztpanel requires a `zerotier-one` controller already running on the
host (or reachable over the network) — it does not run its own
ZeroTier daemon.

## Prebuilt image

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

## Available tags

| Tag              | Meaning                                  |
|-------------------|---------------------------------------------|
| `latest`          | Latest build of the default branch (`main`) |
| `<major>.<minor>` | Latest patch release of that minor version  |
| `<version>`       | An exact release, e.g. `1.0.0`              |

See [Configuration](../configuration) for the environment variables
ztpanel accepts.
