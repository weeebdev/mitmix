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

## Toolchain
- Hub: **Go 1.22+** — NOT currently installed on this machine (no `go` on PATH).
  Install before building hub: `brew install go` (macOS).
- Agents: Python 3.11+, `mitmproxy`, `pocketbase` (PyPI).

## Conventions
- Hub in Go, faithful to Beszel (single binary, embedded PocketBase, `/api/mitm/*` routes).
- Agents in Python, use native mitmproxy addon hooks (NOT subprocess bridging).
- Rule matching runs in the agent; hub is source of truth + realtime distributor.
- All persistent state lives in PocketBase collections (see README/PLAN).
- DO NOT add comments unless asked. Keep files minimal during scaffold phase.

## Build / run
- Hub: `go run .` (after Go installed) — boots PocketBase on :8090.
- Agent: `cd agent && pip install -e . && python mitm_agent.py --hub ws://localhost:8090 --token <node_token>`

## Status
Scaffold phase. PLAN.md holds the phased roadmap (hub skeleton → agent skeleton → auth+rule
sync → flow capture → dashboard).
