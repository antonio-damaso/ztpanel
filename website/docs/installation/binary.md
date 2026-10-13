---
sidebar_position: 2
---

# Prebuilt binary

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

Open it in a browser. On first run, with no users yet, ztpanel serves
a setup page instead of the login screen — pick the admin username
and password there (the page also shows whether it could reach the
ZeroTier controller) and you're logged straight in.

See [Configuration](../configuration) for the environment variables
ztpanel accepts.
