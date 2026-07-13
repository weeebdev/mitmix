# Dashboard

The dashboard is a Svelte 5 app embedded in the hub binary at `/dashboard`.

## Tabs

| Tab | Description |
|-----|-------------|
| **Nodes** | Connected agents, their status and fingerprint. Auto-refreshes on node_up/node_down via WebSocket. |
| **Rules** | CRUD for mitmproxy rules. Drag-and-drop reorder. Live-pushed to agents on save. |
| **Flows** | Captured HTTP flows. Filter by host, method, status. Click to see detail (Overview / Request / Response tabs with headers and body). |
| **Stats** | Aggregated flow statistics: total flows, average/max duration, status code distribution, method breakdown, top hosts, hourly chart. All cards and chart rows are clickable — they navigate to the Flows tab with pre-filled filters. |
| **Queries** | Saved PocketBase filter queries. Run them to filter flows. |
| **Tokens** | Generate and revoke node registration tokens. |

## API

The hub exposes REST endpoints under `/api/mitm/`. All non-ingest endpoints
require PocketBase auth (login via dashboard, token in `Authorization: Bearer` header).

### Auth

Login: `POST /api/collections/_superusers/auth-with-password`
or `POST /api/collections/users/auth-with-password`

Both collections are supported. The dashboard tries superusers first, then users.

### Stats

```
GET /api/mitm/stats
```

Returns:

```json
{
  "total_flows": 42,
  "duration_avg": 234.5,
  "duration_max": 1200,
  "total_req_size": 1048576,
  "total_resp_size": 8388608,
  "status_codes": { "200": 38, "404": 3, "500": 1 },
  "methods": { "GET": 30, "POST": 12 },
  "top_hosts": [
    { "host": "api.example.com", "count": 25 }
  ],
  "hourly": [
    { "hour": "2026-07-13T13:00:00Z", "count": 10 }
  ],
  "success_rate": 95.2
}
```

### Flows

```
GET /api/mitm/flows?host=&method=&status=
POST /api/mitm/flows
GET /api/mitm/flows/:id
```

### CA Certificate

```
GET /api/mitm/ca-cert
```

Returns the mitmproxy CA certificate PEM file. See [agent docs](agent.md#ca-certificate) for install instructions.

## Flow Retention

Old flows and bodies are purged by a background goroutine.
Configure via `FLOW_RETENTION_HOURS` env var (default: 24 hours).

## Live Updates

The dashboard receives real-time updates via WebSocket at `/ws/dash`:
- `node_up` / `node_down` — agent connection status changes (Nodes tab auto-refreshes)
- `flow_created` — new flows from agents (new rows highlighted with green flash)
