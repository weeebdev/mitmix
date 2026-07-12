# mitmix

A **decentralized mitmproxy** — hub/agent architecture. Go hub embeds
PocketBase, Python agents run native mitmproxy addons, all state in
PocketBase. LLM agents interact via MCP.

## Quick start

```sh
git clone https://github.com/weeebdev/mitmix
cd mitmix

# Option A — single container (hub + agent)
docker build -t mitmix -f Dockerfile.single .
docker run -p 8090:8090 -p 8082:8082 mitmix

# Option B — compose (separate containers)
AGENT_TOKEN=test-agent-token docker compose up

# Hub :8090, agent proxy :8082
# Admin: admin@mitmix.local / mitmixadmin123
```

Point your browser proxy at `localhost:8082`, then open `http://localhost:8090`.

## Components

| Component | Language | Role |
|-----------|----------|------|
| **Hub** | Go | PocketBase + WebSocket + MCP + dashboard. Single binary. |
| **Agent** | Python | mitmproxy addons, dials hub via WS, applies rules to live traffic. |
| **Dashboard** | Svelte 5 | Embedded in hub at `/dashboard`. |
| **MCP** | Go (hub) | Streamable HTTP at `/mcp`, 9 tools for LLM agents. |

## Architecture

```
LLM Agent (Claude, etc.)
   │  MCP over HTTP (/mcp)
   ▼
Hub (Go + PocketBase) :8090
   │  WS /ws/agent-connect  ◄── Agent (Python/mitmproxy) :8082
   │  POST /api/mitm/flows  ◄── Agent (batched flow upload)
   │  /dashboard            ►── Browser
   │  /api/mitm/*           ►── REST clients
   │  /mcp                  ►── MCP clients
   │
   └── collections: nodes, rules, flows, flow_bodies, queries, node_tokens
```

Agents dial **out** only — no inbound port needed (NAT-friendly). Rule matching
lives in the agent (low latency); hub is source of truth + realtime distributor.

## Features

### Dashboard (`/dashboard`)

Svelte 5 app embedded in the Go binary.

- **Dashboard** — Beszel-style charts: flow rate (line), status codes (donut),
  top hosts/paths/methods (bar charts)
- **Flows** — paginated list with filtering (host, method, status). Detail panel
  with Overview/Request/Response tabs (headers + truncated body). Export as cURL
  or HAR 1.2. Live streaming via WebSocket.
- **Rules** — create/edit/delete, drag-and-drop reorder, dry-run against test
  flow data, write rule from flow
- **Nodes** — list with status, detail view with agent config info, recent flows
- **Tokens** — generate (crypto/rand base62), copy, revoke
- **Queries** — save named filters, run with one click
- **Alerts** — live stream of node up/down events, webhook support
- **MCP Console** — pick tool, fill args, call `/mcp` directly from UI

### Agent local store

SQLite at `~/.mitm-agent/store.db`. Rules cached on WS connect, loaded from
cache on startup. Flows written immediately to SQLite, uploaded async via
background flush loop. Survives hub outages — queued flows sync on reconnect.
Cleanup deletes synced flows after 24h.

### Selective URL routing

Use `--allow-hosts` / `--ignore-hosts` to control which traffic goes through
the proxy:

```sh
# Only proxy *.example.com and *.myapp.internal
python mitm_agent.py --hub ws://... --token ... \
  --allow-hosts "*.example.com,*.myapp.internal"

# Proxy everything except *.internal
python mitm_agent.py --hub ws://... --token ... \
  --ignore-hosts "*.internal"
```

Glob patterns supported (`*`, `?`, `*.example.com`).

### MCP server (`POST /mcp`)

Streamable HTTP transport, PB Bearer auth required. 9 tools:

| Tool | Description |
|------|-------------|
| `list_nodes` | List all agents with live status |
| `get_node` | Node details including flow count |
| `list_rules` | List rules (optional `enabled` filter) |
| `create_rule` | Create rule (action, match/spec JSON, priority, enabled) |
| `update_rule` | Partial update by rule_id |
| `delete_rule` | Delete by rule_id |
| `toggle_rule` | Enable/disable by rule_id |
| `list_flows` | List flows with filters (host, path, method, status, limit, offset) |
| `get_flow` | Full flow detail including request/response bodies |

```json
POST /mcp
Authorization: Bearer <pb_admin_token>
Content-Type: application/json

{"jsonrpc":"2.0","method":"tools/call","params":{"name":"list_nodes","arguments":{}},"id":"1"}
```

### Rule actions

9 actions supported by the agent's rule engine:

| Action | Description |
|--------|-------------|
| `record` | Capture flow (always-on default) |
| `intercept` | Halt flow for manual review |
| `drop` | Silently drop the request/response |
| `redirect` | Rewrite request URL |
| `modify_headers` | Add/remove/modify HTTP headers |
| `modify_body` | Replace response body content |
| `copy_request` | Clone request to a secondary target |
| `replicate` | Send request to additional endpoints |
| `rewrite` | Modify request host/path components |

Glob matching on `match.host`, `match.path`, `match.method`. Rules evaluated
in priority order (lower number = higher precedence).

### WebSocket protocol (`/ws/agent-connect`)

Agents authenticate via `X-Token` header on WS upgrade:

1. Hub sends `auth_challenge` with nonce
2. Agent responds `auth_response` with fingerprint
3. Hub sends `rules` snapshot (full list)
4. On rule CRUD: hub broadcasts `rule_upsert` or `rule_delete` deltas
5. Hub sends PING every 30s, agent responds PONG

Dashboard WS at `/ws/dash` pushes `flow_created`, `node_up`, `node_down`,
`alert` events in real time.

## API reference

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/ws/agent-connect` | X-Token | WS upgrade for agents |
| GET | `/ws/dash` | — | Dashboard live streaming |
| POST | `/api/mitm/flows` | X-Token | Batch flow ingest (+ body capture, trunc 100KB) |
| GET | `/api/mitm/flows` | PB | List flows (?host=, ?method=, ?status=, ?limit=, ?offset=) |
| GET | `/api/mitm/flows/{id}` | PB | Flow detail with bodies |
| GET | `/api/mitm/flows/{id}/curl` | PB | Export as cURL command (plain text) |
| GET | `/api/mitm/flows/{id}/har` | PB | Export as HAR 1.2 JSON |
| GET | `/api/mitm/stats` | PB | Dashboard stats (flow rate, status breakdown, top hosts/paths/methods) |
| GET | `/api/mitm/rules` | PB | List rules |
| POST | `/api/mitm/rules` | PB | Create rule (validates action, WS broadcast) |
| PUT | `/api/mitm/rules/{id}` | PB | Update rule |
| DELETE | `/api/mitm/rules/{id}` | PB | Delete rule |
| POST | `/api/mitm/rules/reorder` | PB | Batch-reorder by priority |
| POST | `/api/mitm/rules/dry-run` | PB | Test rule `{rule, flow}` → `{matched, valid, summary}` |
| GET | `/api/mitm/nodes` | PB | List nodes |
| GET | `/api/mitm/tokens` | PB | List node tokens |
| POST | `/api/mitm/tokens` | PB | Generate token (crypto/rand base62) |
| DELETE | `/api/mitm/tokens/{id}` | PB | Revoke token |
| GET | `/api/mitm/queries` | PB | List saved queries |
| POST | `/api/mitm/queries` | PB | Create saved query |
| DELETE | `/api/mitm/queries/{id}` | PB | Delete query |
| GET | `/api/mitm/queries/{id}/run` | PB | Run query, return matching flows |
| GET | `/api/mitm/alerts` | PB | List recent alerts (100 most recent) |
| POST | `/mcp` | Bearer | MCP Streamable HTTP (JSON-RPC 2.0) |
| GET | `/dashboard/{path...}` | — | Svelte dashboard (login page if unauthed) |

PB auth = PocketBase superuser token via `Authorization: Bearer <token>`.
X-Token = node token from `node_tokens` collection via `X-Token` header.

## Layout

```
.
├── main.go                      # Hub entry point
├── internal/hub/
│   ├── hub.go                   # Routes, middleware, WS manager
│   ├── ws.go                    # WS keepalive, broadcasting
│   ├── agent_connect.go         # WS upgrade + auth handshake
│   ├── flows.go                 # Flow ingest, list, detail, pagination
│   ├── rules.go                 # Rule CRUD, reorder, dry-run, hooks
│   ├── tokens.go                # Token management
│   ├── alerts.go                # Alert ring buffer, webhook, WS push
│   ├── export.go                # cURL & HAR export endpoints
│   ├── stats.go                 # Dashboard stats aggregation
│   ├── queries.go               # Saved query CRUD + run
│   ├── retention.go             # Flow TTL cleanup
│   ├── mcp.go                   # MCP server (9 tools)
│   ├── dashboard.go             # Svelte SPA handler
│   ├── migrations/              # PB collection definitions
│   ├── hub_test.go              # Integration tests
│   └── site/                    # Svelte 5 dashboard source
├── agent/
│   ├── mitm_agent.py            # Agent entrypoint
│   ├── ws_client.py             # Hub WS client
│   ├── local_store.py           # SQLite local persistence
│   ├── addons/
│   │   ├── rule_engine.py       # 9 rule actions + glob matching
│   │   └── flow_capture.py      # Async batch upload from SQLite
│   └── tests/
├── compose.yaml                 # Hub + agent Docker
├── Dockerfile.hub               # Multi-stage hub build
├── Dockerfile.single            # Single container (hub + agent)
├── agent/Dockerfile.agent       # Python slim
├── supervisord.conf             # Supervisor config for single container
├── .github/workflows/
│   ├── test.yml                 # CI: Go + Python tests, lint
│   └── release.yml              # CD: buildx multi-arch push to ghcr.io
├── Makefile                     # Build/push targets
├── flake.nix                    # Nix dev shell
├── AGENTS.md                    # AI coding agent context
└── PLAN.md                      # Build roadmap
```

## PocketBase collections

| Collection | Fields |
|------------|--------|
| `nodes` | label, token, fingerprint, status (up/down), address, version, last_seen |
| `rules` | node (* = all), priority, action, match (JSON), spec (JSON), enabled |
| `flows` | node, captured_at, method, host, path, status_code, req_size, resp_size, duration_ms, tags, req_headers, resp_headers |
| `flow_bodies` | flow (relation), direction (req/resp), body (text), size |
| `queries` | name, filter (JSON DSL), node, owner |
| `node_tokens` | label, token |

## Dev

```sh
nix develop                    # enter dev shell
go run . serve                 # hub on :8090
cd agent && pip install -e .  # agent deps
python mitm_agent.py --hub ws://localhost:8090 --token test-agent-token

# tests
cd agent && python -m pytest
go test ./...
```

## Deploy options

### Docker compose (recommended for dev)

```sh
AGENT_TOKEN=test-agent-token docker compose up -d
```

Hub on `:8090`, agent proxy on `:8082`.

### Single container (for production / easy deploy)

```sh
docker build -t ghcr.io/weeebdev/mitmix -f Dockerfile.single .
docker run -p 8090:8090 -p 8082:8082 \
  -e AGENT_TOKEN=my-token \
  -e HUB_ADMIN_PASSWORD=strongpass \
  ghcr.io/weeebdev/mitmix
```

### GitHub Container Registry (CI)

Tagged releases are published to `ghcr.io/weeebdev/mitmix` via GitHub Actions.
Multi-arch: `linux/amd64`, `linux/arm64`.

```sh
docker pull ghcr.io/weeebdev/mitmix:v0.1.0
```

## CI/CD

| Workflow | Trigger | What it does |
|----------|---------|-------------|
| `test.yml` | Push/PR to main | `go test ./...`, `python -m pytest`, `ruff check` |
| `release.yml` | Tag `v*` | Buildx multi-arch push to `ghcr.io/weeebdev/mitmix{,-hub,-agent}` |

## Env vars

| Variable | Default | Description |
|----------|---------|-------------|
| `HUB_ADMIN_EMAIL` | `admin@mitmix.local` | Admin email for auto-provision |
| `HUB_ADMIN_PASSWORD` | `mitmixadmin123` | Admin password |
| `FLOW_RETENTION_HOURS` | `24` | Flow TTL in hours |
| `ALERT_WEBHOOK_URL` | — | POST alert JSON to this URL |
| `AGENT_TOKEN` | — | Token for agent to authenticate with hub |
