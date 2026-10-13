---
sidebar_position: 3
---

# From source

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
a setup page instead of the login screen — pick the admin username
and password there (the page also shows whether it could reach the
ZeroTier controller) and you're logged straight in.

See [Configuration](../configuration) for the environment variables
ztpanel accepts.
