# AGENTS.md — mitm-decentralized

## What this is
A decentralized mitmproxy built on the Beszel hub/agent model. Central PocketBase
stores all state (nodes, rules, queries, flows). Python mitmproxy agents dial the
hub, pull rules, apply them to live traffic, and stream captured flows back.

## Components
- **Hub** — Go binary embedding PocketBase. WebSocket at `/ws/agent-connect`, REST
  at `/api/mitm/*`. `main.go` + `internal/`.
- **Agent** — Python, native mitmproxy addons. `agent/mitm_agent.py` runs mitmproxy
  with `addons/rule_engine.py` + `addons/flow_capture.py`, and `agent/ws_client.py`
  keeps the hub WebSocket alive.
- **Dashboard** — Svelte 5 app embedded in hub at `/dashboard`. Login page, flow
  list/detail (Overview/Request/Response tabs), rules (CRUD + drag-and-drop
  reorder), nodes, token management.

## Toolchain
- Hub: **Go 1.22+**. Use `nix develop` (flake provides go_1_25).
- Agents: Python 3.11+, `mitmproxy`, `pocketbase`, `aiohttp` (PyPI).
- Deploy: `docker compose up` (`compose.yaml` + `Dockerfile.hub` /
  `agent/Dockerfile.agent`).

## Conventions
- Hub in Go, faithful to Beszel (single binary, embedded PocketBase).
- Agents in Python, native mitmproxy addon hooks (NOT subprocess bridging).
- Rule matching runs in the agent; hub is source of truth + realtime distributor.
- All persistent state in PocketBase collections (nodes, rules, flows,
  flow_bodies, queries, node_tokens).
- Agent uses port-mapped bridge network in compose (agent connects via `ws://hub:8090`).
- NO comments unless asked.

## Build / run
- Hub: `nix develop && go run . serve` — PocketBase on `:8090`.
  Auto-creates admin: `admin@mitm.local` / `mitmadmin123`.
- Agent: `docker compose up -d` or manually:
  `cd agent && pip install -e . && python mitm_agent.py --hub ws://localhost:8090 --token <node_token>`
- Agent mitmproxy listens on `:8082` (port 8080 taken by OrbStack).

## Dashboard
Svelte 5 at `internal/hub/site/`. Rebuild:
```sh
cd internal/hub/site && npm install && npm run build
```
Then rebuild Go binary. Assets embedded via `//go:embed`.

## Tests
- Agent: `nix develop`, `cd agent`, `python -m venv .venv`, `. .venv/bin/activate`,
  `pip install -e '.[test]'`, `python -m pytest`
- Hub: `go test ./...`

## Auth
- Dashboard login: POST `/api/collections/_superusers/auth-with-password`
  (identity/password). Token stored in localStorage `pb_token`.
- Agent auth: X-Token header on WS upgrade. No PB auth required.
- Flow ingest: X-Token header validation against `node_tokens`.

## API endpoints
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/ws/agent-connect` | X-Token | WS upgrade for agents |
| POST | `/api/mitm/flows` | X-Token | Batch flow ingest (+ body capture) |
| GET | `/api/mitm/flows` | PB | List flows (?host=, ?method=, ?status=) |
| GET | `/api/mitm/flows/{id}` | PB | Flow detail with bodies |
| GET | `/api/mitm/rules` | PB | List rules |
| POST | `/api/mitm/rules` | PB | Create rule (live push to agents) |
| PUT | `/api/mitm/rules/{id}` | PB | Update rule |
| DELETE | `/api/mitm/rules/{id}` | PB | Delete rule |
| POST | `/api/mitm/rules/reorder` | PB | Batch-reorder rules (priority) |
| GET | `/api/mitm/nodes` | PB | List nodes |
| GET | `/api/mitm/tokens` | PB | List node tokens |
| POST | `/api/mitm/tokens` | PB | Generate token (crypto/rand base62) |
| DELETE | `/api/mitm/tokens/{id}` | PB | Revoke token |
| GET | `/dashboard/{path...}` | — | Svelte dashboard (unauthed, login page) |

## Rule actions
9 actions: `record`, `intercept`, `drop`, `redirect`, `modify_headers`,
`modify_body`, `copy_request`, `replicate`, `rewrite`. Glob matching on host/path,
priority-sorted evaluation.

## Flow body capture
Agent extracts request/response headers + body (truncated at 100KB) in
`response()` hook. Bodies stored in `flow_bodies` collection. Dashboard shows
headers + body content in tabs.

## Status
Phase 1 (hub skeleton) + Phase 2 (agent skeleton) + Phase 3 (auth+rule sync) +
Phase 4 (flow capture) + Phase 5 (dashboard) complete. Phase 6 (MCP) next.
