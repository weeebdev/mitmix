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
- Agent: `docker compose up --build -d` or manually:
  `cd agent && pip install -e . && python mitm_agent.py --hub ws://localhost:8090 --token <node_token>`
- Agent mitmproxy listens on `:8082`.
- Always use `docker compose up --build -d` (NOT `docker compose up -d --build <service>`) for full rebuild.

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

## Docs
- `docs/agent.md` — agent setup, CLI flags, cert install, tailscale
- `docs/dashboard.md` — dashboard tabs, API, live updates, retention
- `docs/deployment.md` — Docker compose, env vars, TLS, persistence

## Auth
- Dashboard login: POST `/api/collections/_superusers/auth-with-password`
  or `/api/collections/users/auth-with-password`. Dashboard tries superusers first, then users.
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
| GET | `/api/mitm/stats` | PB | Aggregated flow stats |
| GET | `/api/mitm/ca-cert` | — | Download mitmproxy CA cert PEM |
| GET | `/api/mitm/queries` | PB | List saved queries |
| POST | `/api/mitm/queries` | PB | Create query |
| DELETE | `/api/mitm/queries/{id}` | PB | Delete query |
| GET | `/api/mitm/queries/{id}/run` | PB | Execute query |
| GET | `/dashboard/{path...}` | — | Svelte dashboard (unauthed, login page) |

## Rule actions
9 actions: `record`, `intercept`, `drop`, `redirect`, `modify_headers`,
`modify_body`, `copy_request`, `replicate`, `rewrite`. Glob matching on host/path,
priority-sorted evaluation.

## Flow body capture
Agent extracts request/response headers + body (truncated at 100KB) in
`response()` hook. Bodies stored in `flow_bodies` collection. Dashboard shows
headers + body content in tabs.

## Git operations
- **Rebasing across branches:** Always compare local branch against upstream tracking branch
  (`git log --oneline upstream/<branch>..<branch>`) to verify no commits were dropped.
  Rebase can silently skip commits via conflict resolution (rerere) — always diff the tree
  afterward against the source branch.
- **Cherry-pick vs rebase:** When combining branches with divergent histories (e.g., feature
  branch vs rebrand branch), cherry-pick the smaller set of commits onto the larger feature
  branch. This avoids the "empty commit" problem where rerere resolves conflicts in favor
  of HEAD and discards incoming changes.
- Use `git worktree` for any file edits per global AGENTS.md.

## Status
Phase 1-5 complete, committed, pushed to `upstream/mega-features`.
- Stats dashboard with clickable charts (Stats.svelte + GET /api/mitm/stats)
- Multi-collection login (_superusers + users)
- Agent alerts via WS: ws.go broadcasts node_up/node_down, Nodes.svelte auto-refreshes
- compose.yaml hub env: FLOW_RETENTION_HOURS, HUB_TLS_CERT/KEY, HUB_HTTPS

## Tailscale exit node (next)
Agent can join a Tailscale tailnet as exit node, so non-agent Tailscale devices
route traffic through mitmproxy. Implemented via:
- `--tailscale` flag on agent CLI
- `tailscale` binary installed in agent Dockerfile
- On startup: `tailscale up --auth-key=TS_AUTH_KEY --advertise-routes=0.0.0.0/0,::/0`
- mitmproxy captures all traffic routed through the exit node

Env vars for Tailscale agent:
- `TS_AUTH_KEY` — Tailscale auth key (reusable, pre-approved)
- `TS_HOSTNAME` — optional hostname for the agent node
