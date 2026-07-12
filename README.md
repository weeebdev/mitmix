# mitm-decentralized

A **decentralized mitmproxy** on the Beszel hub/agent model. Go hub embeds
PocketBase, Python agents run native mitmproxy addons, all state in
PocketBase collections. LLM agents interact via MCP.

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
Hub (Go + PocketBase)
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

## Quick start

```sh
git clone https://github.com/adil/mitm-decentralized
cd mitm-decentralized

# Hub
nix develop && go run . serve
# -> PocketBase on :8090, admin admin@mitm.local / mitmadmin123

# Agent (separate terminal)
cd agent && pip install -e . && python mitm_agent.py --hub ws://localhost:8090 --token test-agent-token

# Or Docker
docker compose up
# Hub :8090, agent proxy :8082
```

## Features

### Dashboard (`/dashboard`)

Svelte 5 app embedded in the Go binary.

- **Login** — PB superuser auth, token stored in localStorage
- **Flows** — list with filtering (host, method, status), detail panel with
  Overview/Request/Response tabs (headers + truncated body)
- **Rules** — create/edit/delete, drag-and-drop reorder, write rule from flow
- **Nodes** — list with status, detail view with recent flows
- **Tokens** — generate (crypto/rand base62), copy, revoke
- **Copy as cURL** — from flow detail

### Agent local store

All agents store rules + flows in local SQLite (`~/.mitm-agent/store.db`):

- Rules cached on WS connect, loaded from cache on startup
- Flows written to SQLite immediately, uploaded to hub asynchronously
- Survives hub outages — queued flows sync on reconnect
- Cleanup: synced flows purged after 24h

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

Usage with any MCP client (Claude Code, etc.):

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

### API reference

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/ws/agent-connect` | X-Token | WS upgrade for agents |
| POST | `/api/mitm/flows` | X-Token | Batch flow ingest with bodies |
| GET | `/api/mitm/flows` | PB | List flows (?host=, ?method=, ?status=) |
| GET | `/api/mitm/flows/{id}` | PB | Flow detail with bodies |
| GET | `/api/mitm/rules` | PB | List rules |
| POST | `/api/mitm/rules` | PB | Create rule (WS broadcast) |
| PUT | `/api/mitm/rules/{id}` | PB | Update rule |
| DELETE | `/api/mitm/rules/{id}` | PB | Delete rule |
| POST | `/api/mitm/rules/reorder` | PB | Batch-reorder rules |
| GET | `/api/mitm/nodes` | PB | List nodes |
| GET | `/api/mitm/tokens` | PB | List node tokens |
| POST | `/api/mitm/tokens` | PB | Generate token |
| DELETE | `/api/mitm/tokens/{id}` | PB | Revoke token |
| POST | `/mcp` | Bearer | MCP Streamable HTTP |

PB auth = PocketBase superuser token via `Authorization: Bearer <token>`.

## Layout

```
.
├── main.go                      # Hub entry point
├── internal/hub/
│   ├── hub.go                   # Routes, middleware, WS manager
│   ├── ws.go                    # WS keepalive, broadcasting
│   ├── agent_connect.go         # WS upgrade + auth handshake
│   ├── flows.go                 # Flow ingest, list, detail
│   ├── rules.go                 # Rule CRUD, reorder, hooks
│   ├── tokens.go                # Token management
│   ├── mcp.go                   # MCP server (9 tools)
│   ├── dashboard.go             # Svelte SPA handler
│   ├── migrations/              # PB collection definitions
│   └── site/                    # Svelte 5 dashboard source
├── agent/
│   ├── mitm_agent.py            # Agent entrypoint
│   ├── ws_client.py             # Hub WS client
│   ├── local_store.py           # SQLite local persistence
│   ├── pb_client.py             # PocketBase REST wrapper
│   ├── addons/
│   │   ├── rule_engine.py       # 9 rule actions + glob matching
│   │   └── flow_capture.py      # Async batch upload from SQLite
│   └── tests/
├── compose.yaml                 # Hub + agent Docker
├── Dockerfile.hub               # Multi-stage (node → go → alpine)
├── agent/Dockerfile.agent       # Python slim
├── flake.nix                    # Nix dev shell
├── AGENTS.md                    # AI coding agent context
└── PLAN.md                      # Build roadmap
```

## PocketBase collections

| Collection | Fields |
|------------|--------|
| `nodes` | label, token, fingerprint, status (up/down), address, version, last_seen |
| `rules` | node (\* = all), priority, action, match (JSON), spec (JSON), enabled |
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

## Deploy

```sh
AGENT_TOKEN=<token> docker compose up
# Hub :8090, agent proxy :8082
```

The agent uses `network_mode: host` in compose for DNS resolution.
mitmproxy listens on `:8082` (port 8080 reserved by OrbStack).
