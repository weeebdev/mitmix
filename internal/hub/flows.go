package hub

import (
	"encoding/json"
	"log"

	"github.com/pocketbase/pocketbase/core"
)

type FlowRecord struct {
	Node       string `json:"node"`
	Timestamp  string `json:"timestamp"`
	Method     string `json:"method"`
	Host       string `json:"host"`
	Path       string `json:"path"`
	StatusCode int    `json:"status_code"`
	ReqSize    int    `json:"req_size"`
	RespSize   int    `json:"resp_size"`
	DurationMs int    `json:"duration_ms"`
	Tags       []string `json:"tags"`
}

type IngestRequest struct {
	Flows []FlowRecord `json:"flows"`
}

func (h *Hub) handleIngestFlows(e *core.RequestEvent) error {
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

func (h *Hub) handleListFlows(e *core.RequestEvent) error {
	records, err := h.FindRecordsByFilter("flows", "", "", 100, 0)
	if err != nil {
		log.Printf("flows query error: %v", err)
		return e.InternalServerError("query failed", err)
	}
	return e.JSON(200, records)
}

func (h *Hub) handleListNodes(e *core.RequestEvent) error {
	records, err := h.FindRecordsByFilter("nodes", "", "", 100, 0)
	if err != nil {
		log.Printf("nodes query error: %v", err)
		return e.InternalServerError("query failed", err)
	}
	return e.JSON(200, records)
}
