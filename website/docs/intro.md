---
slug: /
sidebar_position: 1
---

# Introduction

**ztpanel** is a lightweight web UI for a self-hosted [ZeroTier](https://www.zerotier.com/)
controller: a single Go binary (REST API + embedded Vue 3 / PrimeVue
frontend) that talks to your existing `zerotier-one` controller over
its local HTTP API.

It does **not** bundle or run its own ZeroTier daemon, and it does
**not** need an external database — everything ztpanel needs (users,
session key, organizations, activity log) is stored on disk, which is
why a single-binary deployment with no external database is
preferred.

## Comparison

|                        | ztncui                                    | ztnet                                              | ztpanel                                             |
|------------------------|--------------------------------------------|------------------------------------------------------|------------------------------------------------------|
| Deployment             | Single binary/container                   | Needs a database (Postgres)                         | Single binary, no database                          |
| ZeroTier daemon        | Internal (bundled in the same container)  | External (separate sidecar container)               | External (connects to a controller you already run) |
| License                | GPLv3                                      | GPLv3                                                | MIT                                                   |
| Organizations          | No                                          | Yes                                                   | Yes                                                    |
| Activity log           | No                                          | Yes                                                   | Yes                                                    |
| Auth token editing     | No (documented as missing)                 | Yes                                                   | Yes                                                    |

Ready to try it? Head to [Installation](./installation/docker).
