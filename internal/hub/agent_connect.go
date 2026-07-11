package hub

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *Hub) handleAgentConnect(e *core.RequestEvent) error {
	token := e.Request.Header.Get("X-Token")
	if token == "" {
		return e.BadRequestError("missing X-Token header", nil)
	}

	tokenRec, err := h.FindFirstRecordByFilter("node_tokens", "token = {:token}", dbx.Params{"token": token})
	if err != nil {
		return e.UnauthorizedError("invalid token", nil)
	}

	conn, err := upgrader.Upgrade(e.Response, e.Request, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return nil
	}

	ac := &AgentConn{
		Conn:    conn,
		Token:   token,
		closeCh: make(chan struct{}),
	}

	if err := ac.SendJSON(map[string]any{
		"action": "auth_challenge",
		"data":   map[string]string{"nonce": "mitm-hub-v1"},
	}); err != nil {
		conn.Close()
		return nil
	}

	_, resp, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return nil
	}

	var parsed map[string]any
	if err := json.Unmarshal(resp, &parsed); err != nil {
		conn.Close()
		return nil
	}
	if parsed["action"] != "auth_response" {
		conn.Close()
		return nil
	}

	data, _ := parsed["data"].(map[string]any)
	if fp, ok := data["fingerprint"].(string); ok {
		ac.Node = fp
	}

	h.ws.Register(token, ac)
	h.upsertNode(tokenRec, ac)

	go h.listenAgentWS(ac)

	return nil
}

func (h *Hub) upsertNode(tokenRec *core.Record, ac *AgentConn) {
	col, err := h.FindCollectionByNameOrId("nodes")
	if err != nil {
		log.Printf("nodes collection not found: %v", err)
		return
	}

	records, err := h.FindRecordsByFilter("nodes", "token = {:token}", "", 1, 0, dbx.Params{"token": ac.Token})
	if err == nil && len(records) > 0 {
		rec := records[0]
		rec.Set("status", "up")
		rec.Set("last_seen", time.Now().UTC().Format(time.RFC3339))
		rec.Set("fingerprint", ac.Node)
		if err := h.Save(rec); err != nil {
			log.Printf("node update error: %v", err)
		}
		return
	}

	rec := core.NewRecord(col)
	name := tokenRec.GetString("label")
	if name == "" {
		name = "agent-" + ac.Token[:8]
	}
	rec.Set("name", name)
	rec.Set("token", ac.Token)
	rec.Set("fingerprint", ac.Node)
	rec.Set("status", "up")
	rec.Set("last_seen", time.Now().UTC().Format(time.RFC3339))
	rec.Set("version", "mitm-agent-1.0")
	if err := h.Save(rec); err != nil {
		log.Printf("node create error: %v", err)
	}
}
