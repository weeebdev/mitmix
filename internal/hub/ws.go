package hub

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pocketbase/pocketbase/core"
)

type AgentConn struct {
	Conn    *websocket.Conn
	Token   string
	Node    string
	mu      sync.Mutex
	closeCh chan struct{}
}

func (a *AgentConn) SendJSON(msg any) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.Conn.WriteJSON(msg)
}

func (a *AgentConn) Close() {
	close(a.closeCh)
}

type WSManager struct {
	hub  *Hub
	mu   sync.RWMutex
	conn map[string]*AgentConn
}

func NewWSManager(h *Hub) *WSManager {
	return &WSManager{
		hub:  h,
		conn: make(map[string]*AgentConn),
	}
}

func (m *WSManager) Register(token string, ac *AgentConn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if old, ok := m.conn[token]; ok {
		old.Conn.Close()
	}
	m.conn[token] = ac
	log.Printf("agent %s connected (%d total)", token[:8]+"...", len(m.conn))
}

func (m *WSManager) Unregister(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.conn, token)
	log.Printf("agent %s disconnected (%d remaining)", token[:8]+"...", len(m.conn))
}

func (m *WSManager) Broadcast(msg any) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, ac := range m.conn {
		if err := ac.SendJSON(msg); err != nil {
			log.Printf("broadcast error: %v", err)
		}
	}
}

func (m *WSManager) PushRules(ac *AgentConn, rules []*core.Record) {
	type ruleMsg struct {
		ID       string `json:"id"`
		Node     string `json:"node"`
		Priority int    `json:"priority"`
		Action   string `json:"action"`
		Enabled  bool   `json:"enabled"`
		Match    any    `json:"match"`
		Spec     any    `json:"spec"`
	}
	rulesList := make([]ruleMsg, 0, len(rules))
	for _, r := range rules {
		var match, spec any
		json.Unmarshal([]byte(r.GetString("match")), &match)
		json.Unmarshal([]byte(r.GetString("spec")), &spec)
		rulesList = append(rulesList, ruleMsg{
			ID:       r.GetString("id"),
			Node:     r.GetString("node"),
			Priority: r.GetInt("priority"),
			Action:   r.GetString("action"),
			Enabled:  r.GetBool("enabled"),
			Match:    match,
			Spec:     spec,
		})
	}
	ac.SendJSON(map[string]any{
		"action": "rules",
		"data":   rulesList,
	})
}

func (h *Hub) listenAgentWS(ac *AgentConn) {
	defer func() {
		h.ws.Unregister(ac.Token)
		ac.Conn.Close()
	}()

	ac.Conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	ac.Conn.SetPongHandler(func(string) error {
		ac.Conn.SetReadDeadline(time.Now().Add(70 * time.Second))
		return nil
	})

	rules, err := h.FindRecordsByFilter("rules", "enabled = true", "priority", 100, 0)
	if err == nil {
		h.ws.PushRules(ac, rules)
	} else {
		log.Printf("failed to load rules: %v", err)
	}

	for {
		select {
		case <-ac.closeCh:
			return
		default:
		}

		_, msg, err := ac.Conn.ReadMessage()
		if err != nil {
			log.Printf("agent %s read error: %v", ac.Token[:8]+"...", err)
			return
		}

		var parsed map[string]any
		if err := json.Unmarshal(msg, &parsed); err != nil {
			continue
		}

		action, _ := parsed["action"].(string)
		switch action {
		case "pong":
			ac.Conn.SetReadDeadline(time.Now().Add(70 * time.Second))
		case "heartbeat":
			// update last_seen
		}
	}
}
