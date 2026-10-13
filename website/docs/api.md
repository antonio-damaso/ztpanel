---
sidebar_position: 4
---

# MCP / API access

ztpanel exposes a [Model Context Protocol](https://modelcontextprotocol.io/)
endpoint at `POST /mcp`, so an LLM can list/create/update/delete
networks and members directly. Organizations and the activity log are
exposed read-only; membership and org structure stay a web-UI-only
action.

Authentication is a long-lived Bearer API token, not OAuth: ztpanel is
a single, self-hosted binary with no authorization-server
infrastructure of its own, so a static token — generated once in
**Account -> API tokens** — is the simpler fit.

There is only one kind of token, and it authenticates both paths: the
`/api/*` REST API used by the SPA and by scripts, and the `/mcp`
endpoint used by MCP clients. The same `Authorization: Bearer <token>`
header works on either one — it's two protocols over one
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
authorization server — ztpanel isn't one. It works with any MCP
client that can send a static Bearer header (Claude Code, most
self-hosted/CLI clients).

## Token scope and expiry

A token can optionally be **read-only** (list/get only — no creates,
updates or deletes, on either `/api/*` or `/mcp`) and/or set to
**expire** after a number of days. Both are chosen when the token is
created, in **Account -> API tokens**, and can't be changed
afterwards — revoke it and create a new one instead. A read-only
token is the safer default for handing to an LLM that will mostly be
asked to look things up.
