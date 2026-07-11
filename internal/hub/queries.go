package hub

import (
	"encoding/json"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type QueryRequest struct {
	Name   string `json:"name"`
	Filter string `json:"filter"`
}

func (h *Hub) handleListQueries(e *core.RequestEvent) error {
	records, err := h.FindRecordsByFilter("queries", "1=1", "name", 100, 0)
	if err != nil {
		return e.InternalServerError("query failed", err)
	}
	return e.JSON(200, records)
}

func (h *Hub) handleCreateQuery(e *core.RequestEvent) error {
	var req QueryRequest
	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("invalid request", nil)
	}
	if req.Name == "" {
		return e.BadRequestError("name is required", nil)
	}

	col, err := h.FindCollectionByNameOrId("queries")
	if err != nil {
		return e.InternalServerError("collection not found", nil)
	}
	rec := core.NewRecord(col)
	rec.Set("name", req.Name)
	rec.Set("filter", req.Filter)
	if err := h.Save(rec); err != nil {
		return e.InternalServerError("save failed", nil)
	}
	return e.JSON(201, rec)
}

func (h *Hub) handleDeleteQuery(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	rec, err := h.FindRecordById("queries", id)
	if err != nil {
		return e.NotFoundError("query not found", nil)
	}
	if err := h.Delete(rec); err != nil {
		return e.InternalServerError("delete failed", nil)
	}
	return e.JSON(200, map[string]any{"deleted": id})
}

func (h *Hub) handleRunQuery(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	rec, err := h.FindRecordById("queries", id)
	if err != nil {
		return e.NotFoundError("query not found", nil)
	}

	var filterObj map[string]any
	filterStr := rec.GetString("filter")
	if filterStr != "" {
		json.Unmarshal([]byte(filterStr), &filterObj)
	}

	var filters []string
	params := dbx.Params{}

	if host, ok := filterObj["host"].(string); ok && host != "" {
		filters = append(filters, "host ~ {:host}")
		params["host"] = host
	}
	if path, ok := filterObj["path"].(string); ok && path != "" {
		filters = append(filters, "path ~ {:path}")
		params["path"] = path
	}
	if method, ok := filterObj["method"].(string); ok && method != "" {
		filters = append(filters, "method = {:method}")
		params["method"] = method
	}
	if status, ok := filterObj["status"].(float64); ok && status > 0 {
		filters = append(filters, "status_code = {:status}")
		params["status"] = int(status)
	}
	if node, ok := filterObj["node"].(string); ok && node != "" {
		filters = append(filters, "node = {:node}")
		params["node"] = node
	}
	if tag, ok := filterObj["tag"].(string); ok && tag != "" {
		filters = append(filters, "tags ~ {:tag}")
		params["tag"] = tag
	}

	filter := "1=1"
	if len(filters) > 0 {
		filter = strings.Join(filters, " && ")
	}

	records, err := h.FindRecordsByFilter("flows", filter, "-captured_at", 100, 0, params)
	if err != nil {
		return e.InternalServerError("query failed", err)
	}
	return e.JSON(200, records)
}
