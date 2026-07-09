# mitm-decentralized

A **decentralized mitmproxy** built on the Beszel hub/agent model.

- **Hub** (Go, embeds [PocketBase](https://pocketbase.io/)) — central source of truth for all state: nodes, rules, queries, and captured flows. Serves a dashboard and a WebSocket endpoint agents dial out to.
- **Agents** (Python, native [mitmproxy](https://mitmproxy.org/) addons) — one per proxy node. Dial the hub, pull their assigned rules, apply them to live traffic, and stream captured flows back to the hub.
- **MCP server** — the hub also exposes an [MCP](https://modelcontextprotocol.io/) server so LLM agents can query flows, manage rules, inspect nodes, and run saved queries programmatically.

## Architecture

```
Agent (Python/mitmproxy)
   │  WS dial-out (no inbound port; NAT-friendly)
   ▼
Hub (Go + PocketBase)  ──►  collections: nodes, rules, queries, flows, flow_bodies, node_tokens
   │  ├─ auth challenge (mutual auth, Beszel-style)
   │  ├─ initial rule snapshot
   │  ├─ realtime rule deltas (PocketBase realtime)
   │  └─ flow ingest (batched REST)
```

Rule matching happens **in the agent** (close to traffic, low latency). The hub is the
source of truth and realtime distributor.

## Layout

```
.
├── go.mod
├── main.go                 # embeds PocketBase, registers /api/mitm/* routes
├── internal/
│   ├── hub/hub.go          # Hub struct, startup, migration bootstrap
│   ├── pb/migrations/      # collection definitions (Go migrations)
│   ├── ws/agent_ws.go      # WebSocket hub endpoint, auth, rule push
│   └── api/routes.go       # node registration + batch flow ingest
├── agent/
│   ├── pyproject.toml
│   ├── mitm_agent.py       # launches mitmproxy with addons + WS client loop
│   ├── ws_client.py        # dials hub, auth, realtime rule subscription
│   ├── addons/
│   │   ├── rule_engine.py  # applies rules to live flows
│   │   └── flow_capture.py # streams captured flows back to hub
│   └── pb_client.py        # pocketbase SDK wrapper
└── migrations/pocketbase/  # pb_data schema / pb_migrations json
```

## PocketBase collections

- `nodes` — per agent: `name`, `token`, `fingerprint`, `status`, `last_seen`, `version`, `config_rev`.
- `rules` — `node` (`*` = all), `priority`, `match` (host/regex/path/method), `action` (`intercept`|`modify_headers`|`modify_body`|`redirect`|`drop`|`record`), `spec` (JSON), `enabled`.
- `queries` — saved flow filters: `name`, `filter` (JSON DSL), `node`, `owner`.
- `flows` — captured traffic; indexed by node+timestamp; needs retention policy.
- `flow_bodies` — optional separate store for req/resp bodies.
- `node_tokens` — agent registration tokens (Beszel universal-token equivalent).

## Status

Scaffold phase. See `PLAN.md` for the full build plan and phased roadmap.

## Requirements

- Go 1.22+ (hub) — or use the Nix dev shell: `nix develop`
- Python 3.11+ (agents), `mitmproxy`, `pocketbase` (PyPI)
- Docker + Compose for containerized deploy: `docker compose up`

## Dev environment (Nix)

A `flake.nix` provides Go 1.22, Python 3.11, mitmproxy, and docker-compose:

```sh
nix develop        # enter dev shell with go, python, mitmproxy, docker-compose
go run .           # run hub
cd agent && pip install -e .
```

## Deploy (Compose)

`compose.yaml` brings up the hub (PocketBase on :8090) and an agent:

```sh
AGENT_TOKEN=<node_token> docker compose up
```
