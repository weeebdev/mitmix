package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		jsonData := `[
  {
    "name": "nodes",
    "type": "base",
    "system": false,
    "fields": [
      { "name": "name", "type": "text", "required": true, "max": 255 },
      { "name": "token", "type": "text", "required": true, "max": 512 },
      { "name": "fingerprint", "type": "text", "max": 512 },
      { "name": "status", "type": "select", "values": ["up","down","paused"], "maxSelect": 1 },
      { "name": "last_seen", "type": "date" },
      { "name": "version", "type": "text", "max": 32 },
      { "name": "config_rev", "type": "number", "onlyInt": true }
    ],
    "indexes": [
      "CREATE UNIQUE INDEX idx_nodes_name ON nodes (name)",
      "CREATE INDEX idx_nodes_status ON nodes (status)"
    ]
  },
  {
    "name": "rules",
    "type": "base",
    "system": false,
    "fields": [
      { "name": "node", "type": "text", "required": true, "max": 255 },
      { "name": "priority", "type": "number", "onlyInt": true },
      { "name": "match", "type": "json" },
      { "name": "action", "type": "text", "required": true, "max": 32 },
      { "name": "spec", "type": "json" },
      { "name": "enabled", "type": "bool" }
    ],
    "indexes": [
      "CREATE INDEX idx_rules_node ON rules (node)",
      "CREATE INDEX idx_rules_enabled ON rules (enabled)"
    ]
  },
  {
    "name": "queries",
    "type": "base",
    "system": false,
    "fields": [
      { "name": "name", "type": "text", "required": true, "max": 255 },
      { "name": "filter", "type": "json" },
      { "name": "owner", "type": "text", "max": 255 }
    ]
  },
  {
    "name": "flows",
    "type": "base",
    "system": false,
    "fields": [
      { "name": "node", "type": "text", "max": 255 },
      { "name": "captured_at", "type": "date" },
      { "name": "method", "type": "text", "max": 10 },
      { "name": "host", "type": "text", "max": 512 },
      { "name": "path", "type": "text", "max": 2048 },
      { "name": "status_code", "type": "number", "onlyInt": true },
      { "name": "req_content_type", "type": "text", "max": 255 },
      { "name": "resp_content_type", "type": "text", "max": 255 },
      { "name": "req_headers", "type": "json" },
      { "name": "resp_headers", "type": "json" },
      { "name": "req_size", "type": "number", "onlyInt": true },
      { "name": "resp_size", "type": "number", "onlyInt": true },
      { "name": "duration_ms", "type": "number", "onlyInt": true },
      { "name": "tags", "type": "json" },
      { "name": "app_name", "type": "text", "max": 255 },
      { "name": "source_host", "type": "text", "max": 255 }
    ],
    "indexes": [
      "CREATE INDEX idx_flows_node ON flows (node)",
      "CREATE INDEX idx_flows_captured_at ON flows (captured_at)",
      "CREATE INDEX idx_flows_host ON flows (host)",
      "CREATE INDEX idx_flows_app_name ON flows (app_name)",
      "CREATE INDEX idx_flows_source_host ON flows (source_host)"
    ]
  },
  {
    "name": "flow_bodies",
    "type": "base",
    "system": false,
    "fields": [
      { "name": "flow", "type": "text", "max": 255 },
      { "name": "direction", "type": "text", "max": 4 },
      { "name": "body", "type": "editor" },
      { "name": "size", "type": "number", "onlyInt": true }
    ],
    "indexes": [
      "CREATE INDEX idx_flow_bodies_flow ON flow_bodies (flow)"
    ]
  },
  {
    "name": "node_tokens",
    "type": "base",
    "system": false,
    "fields": [
      { "name": "token", "type": "text", "required": true, "max": 512 },
      { "name": "label", "type": "text", "max": 255 }
    ],
    "indexes": [
      "CREATE UNIQUE INDEX idx_node_tokens_token ON node_tokens (token)"
    ]
  }
]`
		return app.ImportCollectionsByMarshaledJSON([]byte(jsonData), false)
	}, func(app core.App) error {
		for _, name := range []string{"nodes", "rules", "queries", "flows", "flow_bodies", "node_tokens"} {
			col, _ := app.FindCollectionByNameOrId(name)
			if col != nil {
				app.Delete(col)
			}
		}
		return nil
	})
}
