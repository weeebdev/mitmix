package hub

import (
	"encoding/json"
	"log"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type FlowRecord struct {
	Node       string   `json:"node"`
	Timestamp  string   `json:"timestamp"`
	Method     string   `json:"method"`
	Host       string   `json:"host"`
	Path       string   `json:"path"`
	StatusCode int      `json:"status_code"`
	ReqSize    int      `json:"req_size"`
	RespSize   int      `json:"resp_size"`
	DurationMs int      `json:"duration_ms"`
	Tags       []string `json:"tags"`
}

type IngestRequest struct {
	Flows []FlowRecord `json:"flows"`
}

func (h *Hub) handleIngestFlows(e *core.RequestEvent) error {
	token := e.Request.Header.Get("X-Token")
	if token == "" {
		return e.UnauthorizedError("missing X-Token", nil)
	}
	_, err := h.FindFirstRecordByFilter("node_tokens", "token = {:token}", dbx.Params{"token": token})
	if err != nil {
		return e.UnauthorizedError("invalid token", nil)
	}

	var req IngestRequest
	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("invalid request body", nil)
	}

	collection, err := h.FindCollectionByNameOrId("flows")
	if err != nil {
		return e.InternalServerError("flows collection not found", nil)
	}

	for _, f := range req.Flows {
		rec := core.NewRecord(collection)
		rec.Set("node", f.Node)
		rec.Set("captured_at", f.Timestamp)
		rec.Set("method", f.Method)
		rec.Set("host", f.Host)
		rec.Set("path", f.Path)
		rec.Set("status_code", f.StatusCode)
		rec.Set("req_size", f.ReqSize)
		rec.Set("resp_size", f.RespSize)
		rec.Set("duration_ms", f.DurationMs)
		tagsJson, _ := json.Marshal(f.Tags)
		rec.Set("tags", string(tagsJson))
		if err := h.Save(rec); err != nil {
			log.Printf("failed to save flow: %v", err)
		}
	}

	return e.JSON(200, map[string]any{"ingested": len(req.Flows)})
}

func (h *Hub) handleGetFlow(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")

	rec, err := h.FindRecordById("flows", id)
	if err != nil {
		return e.NotFoundError("flow not found", nil)
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

	result := map[string]any{
		"id":           rec.Get("id"),
		"node":         rec.Get("node"),
		"captured_at":  rec.Get("captured_at"),
		"method":       rec.Get("method"),
		"host":         rec.Get("host"),
		"path":         rec.Get("path"),
		"status_code":  rec.Get("status_code"),
		"req_headers":  rec.Get("req_headers"),
		"resp_headers": rec.Get("resp_headers"),
		"req_size":     rec.Get("req_size"),
		"resp_size":    rec.Get("resp_size"),
		"duration_ms":  rec.Get("duration_ms"),
		"tags":         rec.Get("tags"),
		"req_body":     reqBody,
		"resp_body":    respBody,
	}

	return e.JSON(200, result)
}

func (h *Hub) handleListFlows(e *core.RequestEvent) error {
	filter := "1=1"
	params := dbx.Params{}
	if node := e.Request.URL.Query().Get("node"); node != "" {
		filter = "node = {:node}"
		params["node"] = node
	}
	records, err := h.FindRecordsByFilter("flows", filter, "", 100, 0, params)
	if err != nil {
		log.Printf("flows query error: %v", err)
		return e.InternalServerError("query failed", err)
	}
	return e.JSON(200, records)
}

func (h *Hub) handleListNodes(e *core.RequestEvent) error {
	records, err := h.FindRecordsByFilter("nodes", "1=1", "", 100, 0)
	if err != nil {
		log.Printf("nodes query error: %v", err)
		return e.InternalServerError("query failed", err)
	}
	return e.JSON(200, records)
}
