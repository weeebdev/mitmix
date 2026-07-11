# AGENTS.md — mitm-decentralized

## What this is
A decentralized mitmproxy built on the Beszel hub/agent model. Central PocketBase stores
all state (nodes, rules, queries, flows). Python mitmproxy agents dial the hub, pull rules,
apply them to live traffic, and stream captured flows back.

## Components
- **Hub** — Go binary embedding PocketBase. Registers `/api/mitm/agent-connect` (agent
  WebSocket) and `/api/mitm/flows` (batch flow ingest). Defined in `main.go` +
  `internal/`.
- **Agent** — Python, native mitmproxy addons. `agent/mitm_agent.py` runs mitmproxy with
  `addons/rule_engine.py` + `addons/flow_capture.py`, and `agent/ws_client.py` keeps the
  hub WebSocket alive.
- **MCP server** — hub exposes an MCP server (over PocketBase collections) so LLM agents can
  query flows, manage rules, inspect nodes, and run saved queries. No separate datastore.

## Toolchain
- Hub: **Go 1.22+**. On this Nix machine use `nix develop` (flake provides go_1_25) instead
  of `brew install go`.
- Agents: Python 3.11+, `mitmproxy`, `pocketbase` (PyPI). Also available via `nix develop`.
- Deploy: `docker compose up` (compose.yaml + Dockerfile.hub / agent/Dockerfile.agent).
- MCP: served by the hub; point an MCP client at the hub's MCP endpoint.

## Conventions
- Hub in Go, faithful to Beszel (single binary, embedded PocketBase, `/api/mitm/*` routes).
- Agents in Python, use native mitmproxy addon hooks (NOT subprocess bridging).
- Rule matching runs in the agent; hub is source of truth + realtime distributor.
- All persistent state lives in PocketBase collections (see README/PLAN).
- DO NOT add comments unless asked. Keep files minimal during scaffold phase.

## Build / run
- Hub: `nix develop && go run . serve` — boots PocketBase on :8090.
  Set `HUB_ADMIN_EMAIL` and `HUB_ADMIN_PASSWORD` to auto-create admin (no prompt).
  Default credentials (no env vars): `admin@mitm.local` / `mitmadmin123`.
- Dashboard: open `http://localhost:8090/dashboard` in a browser.
- Agent: `cd agent && pip install -e . && python mitm_agent.py --hub ws://localhost:8090 --token <node_token>`

## Dashboard
Svelte 5 app at `internal/hub/site/`. Rebuild with:
```sh
cd internal/hub/site && npm install && npm run build
```
Then rebuild the Go binary. The compiled assets are embedded via `//go:embed`.

## Tests
- Agent (Python): from repo root, `nix develop`, then:
  `cd agent && python -m venv .venv && . .venv/bin/activate && pip install -e '.[test]' aiohttp && python -m pytest`
  NOTE: build the venv with the Nix shell's Python (3.13) so native libs
  (cryptography/mitmproxy) match; a 3.11 venv breaks the cryptography binary.
- Hub (Go): `go test ./...` (once Go/hub code lands).

## API endpoints
| Method | Path | Description |
|--------|------|-------------|
| GET | `/ws/agent-connect` | WebSocket upgrade for agents (X-Token header, no PB auth) |
| POST | `/api/mitm/flows` | Batch flow ingest (JSON body: `{"flows": [...]}`) |
| GET | `/api/mitm/flows` | List captured flows |
| POST | `/api/mitm/rules` | Create a new rule (triggers live push to agents) |
| GET | `/api/mitm/rules` | List rules |
| GET | `/api/mitm/nodes` | List agent nodes |

## Status
Phase 1 (hub skeleton) + Phase 2 (agent skeleton) complete. Active development.
PLAN.md holds the remaining phases (auth+rule sync → flow capture → dashboard → MCP).
