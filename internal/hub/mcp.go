package hub

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      any             `json:"id"`
}

type mcpResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  any         `json:"result,omitempty"`
	Error   *mcpError   `json:"error,omitempty"`
	ID      any         `json:"id"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

type mcpToolCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type mcpContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (h *Hub) handleMCP(e *core.RequestEvent) error {
	auth := e.Request.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		return e.JSON(401, mcpResponse{JSONRPC: "2.0", Error: &mcpError{Code: -32001, Message: "Unauthorized"}, ID: nil})
	}
	_, err := h.FindAuthRecordByToken(strings.TrimPrefix(auth, "Bearer "), core.TokenTypeAuth)
	if err != nil {
		return e.JSON(401, mcpResponse{JSONRPC: "2.0", Error: &mcpError{Code: -32001, Message: "Invalid token"}, ID: nil})
	}

	var req mcpRequest
	if err := e.BindBody(&req); err != nil {
		return e.JSON(400, mcpResponse{JSONRPC: "2.0", Error: &mcpError{Code: -32700, Message: "Parse error"}, ID: nil})
	}

	var result any
	var rpcErr *mcpError

	switch req.Method {
	case "initialize":
		result = map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "mitm-decentralized",
				"version": "0.1.0",
			},
		}
	case "ping":
		result = map[string]any{}
	case "tools/list":
		result = map[string]any{"tools": listTools()}
	case "tools/call":
		var params mcpToolCall
		if err := json.Unmarshal(req.Params, &params); err != nil {
			rpcErr = &mcpError{Code: -32602, Message: "Invalid params"}
		} else {
			content, err := h.callTool(params.Name, params.Arguments)
			if err != nil {
				rpcErr = &mcpError{Code: -32603, Message: err.Error()}
			} else {
				result = map[string]any{"content": content}
			}
		}
	default:
		rpcErr = &mcpError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", req.Method)}
	}

	return e.JSON(200, mcpResponse{JSONRPC: "2.0", Result: result, Error: rpcErr, ID: req.ID})
}

func listTools() []mcpTool {
	return []mcpTool{
		{
			Name:        "list_nodes",
			Description: "List all agent nodes with live status",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			Name:        "get_node",
			Description: "Get details of a specific agent node including recent flow count",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"node_id": map[string]any{"type": "string", "description": "Node record ID"},
				},
				"required": []string{"node_id"},
			},
		},
		{
			Name:        "list_rules",
			Description: "List all mitmproxy rules, optionally filter by enabled status",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"enabled": map[string]any{"type": "boolean", "description": "Filter by enabled status"},
				},
			},
		},
		{
			Name:        "create_rule",
			Description: "Create a new mitmproxy rule. action must be one of: record, intercept, drop, redirect, modify_headers, modify_body, copy_request, replicate, rewrite. match is a JSON string with host/path/method globs. spec is a JSON string with action-specific config.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action":   map[string]any{"type": "string", "description": "Rule action"},
					"match":    map[string]any{"type": "string", "description": "JSON match config {\"host\":\"*\",\"path\":\"/api/**\",\"method\":\"*\"}"},
					"spec":     map[string]any{"type": "string", "description": "JSON action spec"},
					"priority": map[string]any{"type": "integer", "description": "Priority (lower = higher precedence)"},
					"enabled":  map[string]any{"type": "boolean", "description": "Whether rule is active"},
					"node":     map[string]any{"type": "string", "description": "Optional node ID (empty = all nodes)"},
				},
				"required": []string{"action"},
			},
		},
		{
			Name:        "update_rule",
			Description: "Update an existing rule. Only provided fields are changed.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"rule_id":  map[string]any{"type": "string", "description": "Rule record ID"},
					"action":   map[string]any{"type": "string", "description": "Rule action"},
					"match":    map[string]any{"type": "string", "description": "JSON match config"},
					"spec":     map[string]any{"type": "string", "description": "JSON action spec"},
					"priority": map[string]any{"type": "integer", "description": "Priority"},
					"enabled":  map[string]any{"type": "boolean", "description": "Active status"},
				},
				"required": []string{"rule_id"},
			},
		},
		{
			Name:        "delete_rule",
			Description: "Delete a rule by ID",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"rule_id": map[string]any{"type": "string", "description": "Rule record ID"},
				},
				"required": []string{"rule_id"},
			},
		},
		{
			Name:        "toggle_rule",
			Description: "Enable or disable a rule",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"rule_id": map[string]any{"type": "string", "description": "Rule record ID"},
					"enabled": map[string]any{"type": "boolean", "description": "True = enable, False = disable"},
				},
				"required": []string{"rule_id", "enabled"},
			},
		},
		{
			Name:        "list_flows",
			Description: "List captured flows with optional filters",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"host":    map[string]any{"type": "string", "description": "Filter by host (substring match)"},
					"path":    map[string]any{"type": "string", "description": "Filter by path (substring match)"},
					"method":  map[string]any{"type": "string", "description": "Filter by HTTP method"},
					"status":  map[string]any{"type": "integer", "description": "Filter by HTTP status code"},
					"limit":   map[string]any{"type": "integer", "description": "Max results (default 50)"},
					"offset":  map[string]any{"type": "integer", "description": "Result offset"},
				},
			},
		},
		{
			Name:        "get_flow",
			Description: "Get full flow details including request/response headers and bodies",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"flow_id": map[string]any{"type": "string", "description": "Flow record ID"},
				},
				"required": []string{"flow_id"},
			},
		},
	}
}

func (h *Hub) callTool(name string, args map[string]any) ([]mcpContent, error) {
	switch name {
	case "list_nodes":
		return h.mcpListNodes(args)
	case "get_node":
		return h.mcpGetNode(args)
	case "list_rules":
		return h.mcpListRules(args)
	case "create_rule":
		return h.mcpCreateRule(args)
	case "update_rule":
		return h.mcpUpdateRule(args)
	case "delete_rule":
		return h.mcpDeleteRule(args)
	case "toggle_rule":
		return h.mcpToggleRule(args)
	case "list_flows":
		return h.mcpListFlows(args)
	case "get_flow":
		return h.mcpGetFlow(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (h *Hub) mcpListNodes(_ map[string]any) ([]mcpContent, error) {
	records, err := h.FindRecordsByFilter("nodes", "1=1", "-last_seen", 100, 0)
	if err != nil {
		return nil, fmt.Errorf("query nodes: %w", err)
	}
	var out []map[string]any
	for _, r := range records {
		out = append(out, map[string]any{
			"id":         r.GetString("id"),
			"label":      r.GetString("label"),
			"status":     r.GetString("status"),
			"address":    r.GetString("address"),
			"version":    r.GetString("version"),
			"last_seen":  r.GetString("last_seen"),
			"created":    r.GetString("created"),
		})
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	return []mcpContent{{Type: "text", Text: string(b)}}, nil
}

func (h *Hub) mcpGetNode(args map[string]any) ([]mcpContent, error) {
	id, _ := args["node_id"].(string)
	if id == "" {
		return nil, fmt.Errorf("node_id is required")
	}
	rec, err := h.FindRecordById("nodes", id)
	if err != nil {
		return nil, fmt.Errorf("node not found: %s", id)
	}
	flowCount, _ := h.FindRecordsByFilter("flows", "node = {:node}", "", 0, 0, dbx.Params{"node": rec.GetString("label")})
	out := map[string]any{
		"id":          rec.GetString("id"),
		"label":       rec.GetString("label"),
		"status":      rec.GetString("status"),
		"address":     rec.GetString("address"),
		"version":     rec.GetString("version"),
		"fingerprint": rec.GetString("fingerprint"),
		"last_seen":   rec.GetString("last_seen"),
		"created":     rec.GetString("created"),
		"updated":     rec.GetString("updated"),
		"flow_count":  len(flowCount),
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	return []mcpContent{{Type: "text", Text: string(b)}}, nil
}

func (h *Hub) mcpListRules(args map[string]any) ([]mcpContent, error) {
	filter := "1=1"
	params := dbx.Params{}
	if enabled, ok := args["enabled"]; ok {
		if v, ok := enabled.(bool); ok {
			filter = "enabled = {:enabled}"
			params["enabled"] = v
		}
	}
	records, err := h.FindRecordsByFilter("rules", filter, "priority", 100, 0, params)
	if err != nil {
		return nil, fmt.Errorf("query rules: %w", err)
	}
	var out []map[string]any
	for _, r := range records {
		out = append(out, map[string]any{
			"id":       r.GetString("id"),
			"node":     r.GetString("node"),
			"action":   r.GetString("action"),
			"priority": r.GetInt("priority"),
			"enabled":  r.GetBool("enabled"),
			"match":    r.GetString("match"),
			"spec":     r.GetString("spec"),
		})
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	return []mcpContent{{Type: "text", Text: string(b)}}, nil
}

func (h *Hub) mcpCreateRule(args map[string]any) ([]mcpContent, error) {
	action, _ := args["action"].(string)
	if action == "" {
		return nil, fmt.Errorf("action is required")
	}
	match, _ := args["match"].(string)
	if match == "" {
		match = `{"host":"*","path":"**","method":"*"}`
	}
	spec, _ := args["spec"].(string)
	if spec == "" {
		spec = "{}"
	}
	priority, _ := args["priority"].(float64)
	enabled, _ := args["enabled"].(bool)
	node, _ := args["node"].(string)

	col, err := h.FindCollectionByNameOrId("rules")
	if err != nil {
		return nil, fmt.Errorf("rules collection not found")
	}
	rec := core.NewRecord(col)
	if node == "" {
		node = "*"
	}
	rec.Set("node", node)
	rec.Set("priority", int(priority))
	rec.Set("action", action)
	rec.Set("match", match)
	rec.Set("spec", spec)
	rec.Set("enabled", enabled)
	if err := h.Save(rec); err != nil {
		return nil, fmt.Errorf("save rule: %w", err)
	}
	b, _ := json.MarshalIndent(map[string]any{
		"id":       rec.GetString("id"),
		"node":     rec.GetString("node"),
		"action":   rec.GetString("action"),
		"priority": rec.GetInt("priority"),
		"enabled":  rec.GetBool("enabled"),
		"match":    rec.GetString("match"),
		"spec":     rec.GetString("spec"),
	}, "", "  ")
	return []mcpContent{{Type: "text", Text: string(b)}}, nil
}

func (h *Hub) mcpUpdateRule(args map[string]any) ([]mcpContent, error) {
	id, _ := args["rule_id"].(string)
	if id == "" {
		return nil, fmt.Errorf("rule_id is required")
	}
	rec, err := h.FindRecordById("rules", id)
	if err != nil {
		return nil, fmt.Errorf("rule not found: %s", id)
	}
	if v, ok := args["action"]; ok {
		rec.Set("action", v)
	}
	if v, ok := args["match"]; ok {
		rec.Set("match", v)
	}
	if v, ok := args["spec"]; ok {
		rec.Set("spec", v)
	}
	if v, ok := args["priority"]; ok {
		rec.Set("priority", int(v.(float64)))
	}
	if v, ok := args["enabled"]; ok {
		rec.Set("enabled", v)
	}
	if err := h.Save(rec); err != nil {
		return nil, fmt.Errorf("save rule: %w", err)
	}
	b, _ := json.MarshalIndent(map[string]any{
		"id":       rec.GetString("id"),
		"node":     rec.GetString("node"),
		"action":   rec.GetString("action"),
		"priority": rec.GetInt("priority"),
		"enabled":  rec.GetBool("enabled"),
		"match":    rec.GetString("match"),
		"spec":     rec.GetString("spec"),
	}, "", "  ")
	return []mcpContent{{Type: "text", Text: string(b)}}, nil
}

func (h *Hub) mcpDeleteRule(args map[string]any) ([]mcpContent, error) {
	id, _ := args["rule_id"].(string)
	if id == "" {
		return nil, fmt.Errorf("rule_id is required")
	}
	rec, err := h.FindRecordById("rules", id)
	if err != nil {
		return nil, fmt.Errorf("rule not found: %s", id)
	}
	if err := h.Delete(rec); err != nil {
		return nil, fmt.Errorf("delete rule: %w", err)
	}
	return []mcpContent{{Type: "text", Text: fmt.Sprintf("Deleted rule %s", id)}}, nil
}

func (h *Hub) mcpToggleRule(args map[string]any) ([]mcpContent, error) {
	id, _ := args["rule_id"].(string)
	if id == "" {
		return nil, fmt.Errorf("rule_id is required")
	}
	enabled, ok := args["enabled"].(bool)
	if !ok {
		return nil, fmt.Errorf("enabled is required")
	}
	rec, err := h.FindRecordById("rules", id)
	if err != nil {
		return nil, fmt.Errorf("rule not found: %s", id)
	}
	rec.Set("enabled", enabled)
	if err := h.Save(rec); err != nil {
		return nil, fmt.Errorf("save rule: %w", err)
	}
	status := "enabled"
	if !enabled {
		status = "disabled"
	}
	return []mcpContent{{Type: "text", Text: fmt.Sprintf("Rule %s %s", id, status)}}, nil
}

func (h *Hub) mcpListFlows(args map[string]any) ([]mcpContent, error) {
	var filters []string
	params := dbx.Params{}

	if host, ok := args["host"].(string); ok && host != "" {
		filters = append(filters, "host ~ {:host}")
		params["host"] = host
	}
	if path, ok := args["path"].(string); ok && path != "" {
		filters = append(filters, "path ~ {:path}")
		params["path"] = path
	}
	if method, ok := args["method"].(string); ok && method != "" {
		filters = append(filters, "method = {:method}")
		params["method"] = method
	}
	if status, ok := args["status"].(float64); ok && status > 0 {
		filters = append(filters, "status_code = {:status}")
		params["status"] = int(status)
	}

	filter := "1=1"
	if len(filters) > 0 {
		filter = strings.Join(filters, " && ")
	}

	limit := 50
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 500 {
			limit = 500
		}
	}
	offset := 0
	if o, ok := args["offset"].(float64); ok && o >= 0 {
		offset = int(o)
	}

	records, err := h.FindRecordsByFilter("flows", filter, "-captured_at", limit, offset, params)
	if err != nil {
		return nil, fmt.Errorf("query flows: %w", err)
	}

	var out []map[string]any
	for _, r := range records {
		out = append(out, map[string]any{
			"id":          r.GetString("id"),
			"node":        r.GetString("node"),
			"method":      r.GetString("method"),
			"host":        r.GetString("host"),
			"path":        r.GetString("path"),
			"status_code": r.GetInt("status_code"),
			"req_size":    r.GetInt("req_size"),
			"resp_size":   r.GetInt("resp_size"),
			"duration_ms": r.GetInt("duration_ms"),
			"captured_at": r.GetString("captured_at"),
			"tags":        r.GetString("tags"),
		})
	}

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("Flows (%d results):\n\n", len(out)))
	for _, f := range out {
		sb.WriteString(fmt.Sprintf("  [%s] %s %s%s -> %d (%dms)\n",
			f["id"].(string)[:8],
			f["method"],
			f["host"],
			f["path"],
			f["status_code"],
			f["duration_ms"],
		))
	}

	return []mcpContent{{Type: "text", Text: sb.String()}}, nil
}

func (h *Hub) mcpGetFlow(args map[string]any) ([]mcpContent, error) {
	id, _ := args["flow_id"].(string)
	if id == "" {
		return nil, fmt.Errorf("flow_id is required")
	}
	rec, err := h.FindRecordById("flows", id)
	if err != nil {
		return nil, fmt.Errorf("flow not found: %s", id)
	}

	bodies, _ := h.FindRecordsByFilter("flow_bodies", "flow = {:flow}", "", 0, 0, dbx.Params{"flow": id})
	var reqBody, respBody string
	for _, b := range bodies {
		dir := b.GetString("direction")
		if dir == "req" {
			reqBody = b.GetString("body")
		} else if dir == "resp" {
			respBody = b.GetString("body")
		}
	}

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("Flow %s\n", id))
	sb.WriteString(fmt.Sprintf("  %s %s%s -> %d (%dms)\n", rec.GetString("method"), rec.GetString("host"), rec.GetString("path"), rec.GetInt("status_code"), rec.GetInt("duration_ms")))
	sb.WriteString(fmt.Sprintf("  Node: %s | Captured: %s\n", rec.GetString("node"), rec.GetString("captured_at")))

	if hdr := rec.GetString("req_headers"); hdr != "" && hdr != "null" {
		sb.WriteString(fmt.Sprintf("  Request Headers: %s\n", hdr))
	}
	if hdr := rec.GetString("resp_headers"); hdr != "" && hdr != "null" {
		sb.WriteString(fmt.Sprintf("  Response Headers: %s\n", hdr))
	}
	if reqBody != "" {
		trunc := reqBody
		if len(trunc) > 2000 {
			trunc = trunc[:2000] + "..."
		}
		sb.WriteString(fmt.Sprintf("  Request Body (%d bytes):\n%s\n", len(reqBody), trunc))
	}
	if respBody != "" {
		trunc := respBody
		if len(trunc) > 2000 {
			trunc = trunc[:2000] + "..."
		}
		sb.WriteString(fmt.Sprintf("  Response Body (%d bytes):\n%s\n", len(respBody), trunc))
	}

	return []mcpContent{{Type: "text", Text: sb.String()}}, nil
}
