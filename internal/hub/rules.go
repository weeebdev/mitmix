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
	records, err := h.FindRecordsByFilter("rules", "", "priority", 100, 0)
	if err != nil {
		log.Printf("rules query error: %v", err)
		return e.InternalServerError("query failed", err)
	}
	return e.JSON(200, records)
}
