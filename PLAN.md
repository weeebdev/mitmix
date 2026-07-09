# PLAN.md — mitm-decentralized

mitmproxy + PocketBase, structured like Beszel (hub + agents).

## 1. Architecture

- **Hub** — Go binary embedding PocketBase. Stores all state (nodes, rules, queries,
  flows) in PocketBase collections. Serves a dashboard and exposes a WebSocket endpoint
  `/api/mitm/agent-connect` for agents to dial out, authenticate, and subscribe to rule
  changes in real time.
- **Agent** — Python process running `mitmproxy` (managed addon/script) on each node.
  On startup it dials the hub WebSocket, authenticates with a node token, and:
  - pulls its assigned **rules** (intercept/modify/redirect/filter),
  - subscribes to live rule updates via PocketBase realtime,
  - runs mitmproxy addons that apply those rules to live traffic,
  - pushes captured **flows** back to the hub (subject to query config).

## 2. Communication (WebSocket push/pull, Beszel-style)

```
Agent ──WS dial-out──> Hub /api/mitm/agent-connect
   <- auth challenge (hub signs, agent verifies)   # mutual auth, like Beszel
   <- rule config (initial snapshot, CBOR/JSON)
   <- realtime rule deltas (PocketBase realtime subscription proxied)
   -> flow records (batched POST to PocketBase REST)
   -> heartbeat / status
```

No inbound port on agents (NAT-friendly), matching Beszel's primary WebSocket mode.

## 3. PocketBase collections

- `nodes` — one per mitmproxy agent: `name`, `token`, `fingerprint`, `status` (up/down),
  `last_seen`, `version`, `config_rev`.
- `rules` — shared or per-node: `node` (or `*` for all), `priority`, `match`
  (host/regex/path/method), `action` (`intercept`|`modify_headers`|`modify_body`|
  `redirect`|`drop`|`record`), `spec` (JSON of changes), `enabled`.
- `queries` — saved search/filter over flows: `name`, `filter` (JSON query DSL), `node`, `owner`.
- `flows` — captured traffic: `node`, `timestamp`, `method`, `host`, `path`, `status_code`,
  `req_headers`, `resp_headers`, `req_size`, `resp_size`, `duration_ms`, `tags`.
  Indexed by node+timestamp; needs retention policy.
- `flow_bodies` — optional separate collection for request/response bodies.
- `node_tokens` — agent registration tokens (Beszel universal-token equivalent).

## 4. Repository layout

```
mitm-decentralized/
├── go.mod
├── main.go                     # embeds PocketBase, registers /api/mitm/* routes + realtime proxy
├── internal/
│   ├── hub/hub.go              # Hub struct, startup, migrations bootstrap
│   ├── pb/migrations/          # collection definitions (Go migration files)
│   ├── ws/agent_ws.go          # WebSocket hub endpoint, auth challenge, rule push
│   └── api/routes.go           # custom REST: node registration, flow ingest batch
├── agent/
│   ├── pyproject.toml
│   ├── mitm_agent.py           # launches mitmproxy with addons, WS client loop
│   ├── ws_client.py            # dials hub, auth, realtime rule subscription
│   ├── addons/
│   │   ├── rule_engine.py      # applies rules to flows (request/response hooks)
│   │   └── flow_capture.py     # streams captured flows back to hub
│   └── pb_client.py            # pocketbase SDK wrapper
├── migrations/
│   └── pocketbase/             # pb_data schema / pb_migrations json
└── README.md
```

## 5. Build phases (scaffold only for now)

1. **Hub skeleton** — Go module, embed PocketBase, `main.go` boots server + registers
   `/api/mitm/agent-connect` and `/api/mitm/flows` (batch ingest). Define collections via
   migration file.
2. **Agent skeleton** — Python pyproject, `mitm_agent.py` that runs mitmproxy
   programmatically (`mitmdump`/`master` API) with placeholder addons; `ws_client.py`
   dials hub with a node token.
3. **Auth + rule sync** — mutual-auth handshake, hub pushes `rules` snapshot, agent
   subscribes via PocketBase realtime (`/api/realtime`) for deltas; `rule_engine.py`
   matches live flows.
4. **Flow capture + storage** — `flow_capture.py` batches flows to hub REST; hub writes to
   `flows` collection; `queries` collection + dashboard list view.
5. **Dashboard** — minimal PocketBase admin views + a small custom UI for nodes, rules,
   live flow stream, saved queries.

## 6. Key decisions

- Go hub stays faithful to Beszel (single binary, embedded PocketBase). Python agents use
  `pocketbase` PyPI SDK + native mitmproxy addon hooks — no subprocess bridging.
- Rule matching lives **in the agent** (close to traffic, low latency); hub is source of
  truth + realtime distributor.
- Flows stored centrally; retention policy needed since this table grows fast.
