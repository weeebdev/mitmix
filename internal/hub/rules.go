package hub

import (
	"encoding/json"
	"log"

	"github.com/pocketbase/pocketbase/core"
)

func (h *Hub) registerRuleHooks() {
	h.OnRecordCreate("rules").BindFunc(func(e *core.RecordEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		h.broadcastRuleDelta("rule_upsert", e.Record)
		return nil
	})

	h.OnRecordUpdate("rules").BindFunc(func(e *core.RecordEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		h.broadcastRuleDelta("rule_upsert", e.Record)
		return nil
	})

	h.OnRecordDelete("rules").BindFunc(func(e *core.RecordEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		h.ws.Broadcast(map[string]any{
			"action": "rule_delete",
			"data":   map[string]string{"id": e.Record.GetString("id")},
		})
		return nil
	})
}

func (h *Hub) broadcastRuleDelta(action string, record *core.Record) {
	var match, spec any
	json.Unmarshal([]byte(record.GetString("match")), &match)
	json.Unmarshal([]byte(record.GetString("spec")), &spec)

	h.ws.Broadcast(map[string]any{
		"action": action,
		"data": map[string]any{
			"id":       record.GetString("id"),
			"node":     record.GetString("node"),
			"priority": record.GetInt("priority"),
			"action":   record.GetString("action"),
			"enabled":  record.GetBool("enabled"),
			"match":    match,
			"spec":     spec,
		},
	})
}

func (h *Hub) handleListRules(e *core.RequestEvent) error {
	records, err := h.FindRecordsByFilter("rules", "1=1", "priority", 100, 0)
	if err != nil {
		log.Printf("rules query error: %v", err)
		return e.InternalServerError("query failed", err)
	}
	return e.JSON(200, records)
}

type CreateRuleRequest struct {
	Node     string `json:"node"`
	Priority int    `json:"priority"`
	Action   string `json:"action"`
	Match    string `json:"match"`
	Spec     string `json:"spec"`
	Enabled  bool   `json:"enabled"`
}

func (h *Hub) handleCreateRule(e *core.RequestEvent) error {
	var req CreateRuleRequest
	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("invalid request", nil)
	}
	if req.Action == "" {
		return e.BadRequestError("action is required", nil)
	}
	if req.Match == "" {
		req.Match = "{}"
	}
	if req.Spec == "" {
		req.Spec = "{}"
	}

	col, err := h.FindCollectionByNameOrId("rules")
	if err != nil {
		return e.InternalServerError("collection not found", nil)
	}

	rec := core.NewRecord(col)
	if req.Node == "" {
		req.Node = "*"
	}
	rec.Set("node", req.Node)
	rec.Set("priority", req.Priority)
	rec.Set("action", req.Action)
	rec.Set("match", req.Match)
	rec.Set("spec", req.Spec)
	rec.Set("enabled", req.Enabled)

	if err := h.Save(rec); err != nil {
		log.Printf("create rule error: %v", err)
		return e.InternalServerError("save failed", nil)
	}

	return e.JSON(201, rec)
}

type UpdateRuleRequest struct {
	Node     *string `json:"node,omitempty"`
	Priority *int    `json:"priority,omitempty"`
	Action   *string `json:"action,omitempty"`
	Match    *string `json:"match,omitempty"`
	Spec     *string `json:"spec,omitempty"`
	Enabled  *bool   `json:"enabled,omitempty"`
}

func (h *Hub) handleUpdateRule(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	rec, err := h.FindRecordById("rules", id)
	if err != nil {
		return e.NotFoundError("rule not found", nil)
	}

	var req UpdateRuleRequest
	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("invalid request", nil)
	}

	if req.Node != nil {
		rec.Set("node", *req.Node)
	}
	if req.Priority != nil {
		rec.Set("priority", *req.Priority)
	}
	if req.Action != nil {
		rec.Set("action", *req.Action)
	}
	if req.Match != nil {
		rec.Set("match", *req.Match)
	}
	if req.Spec != nil {
		rec.Set("spec", *req.Spec)
	}
	if req.Enabled != nil {
		rec.Set("enabled", *req.Enabled)
	}

	if err := h.Save(rec); err != nil {
		log.Printf("update rule error: %v", err)
		return e.InternalServerError("save failed", nil)
	}

	return e.JSON(200, rec)
}

func (h *Hub) handleDeleteRule(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	rec, err := h.FindRecordById("rules", id)
	if err != nil {
		return e.NotFoundError("rule not found", nil)
	}
	if err := h.Delete(rec); err != nil {
		log.Printf("delete rule error: %v", err)
		return e.InternalServerError("delete failed", nil)
	}
	return e.JSON(200, map[string]any{"deleted": id})
}

type ReorderRequest struct {
	IDs []string `json:"ids"`
}

func (h *Hub) handleReorderRules(e *core.RequestEvent) error {
	var req ReorderRequest
	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("invalid request", nil)
	}
	for i, id := range req.IDs {
		rec, err := h.FindRecordById("rules", id)
		if err != nil {
			continue
		}
		rec.Set("priority", (i+1)*10)
		if err := h.Save(rec); err != nil {
			log.Printf("reorder save error: %v", err)
		}
	}
	return e.JSON(200, map[string]any{"ok": true})
}
