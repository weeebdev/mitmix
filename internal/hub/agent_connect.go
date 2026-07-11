package hub

import (
	"encoding/json"
	"log"
	"net/http"

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

	_, err := h.FindFirstRecordByFilter("node_tokens", "token = {:token}", dbx.Params{"token": token})
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

	go h.listenAgentWS(ac)

	return nil
}
