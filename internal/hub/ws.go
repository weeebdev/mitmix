package hub

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 90 * time.Second
	pingInterval   = 30 * time.Second
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

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
	hub     *Hub
	mu      sync.RWMutex
	conn    map[string]*AgentConn
	caCerts map[string]string
}

func NewWSManager(h *Hub) *WSManager {
	return &WSManager{
		hub:     h,
		conn:    make(map[string]*AgentConn),
		caCerts: make(map[string]string),
	}
}

func (m *WSManager) StoreCACert(token, pem string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.caCerts[token] = pem
}

func (m *WSManager) GetCACert() (string, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for token, pem := range m.caCerts {
		if pem != "" {
			return token, pem
		}
	}
	return "", ""
}

func (m *WSManager) Register(token string, ac *AgentConn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if old, ok := m.conn[token]; ok {
		old.Conn.Close()
	}
	m.conn[token] = ac
	log.Printf("agent %s connected (%d total)", token[:8]+"...", len(m.conn))
	if m.hub != nil {
		records, err := m.hub.FindRecordsByFilter("nodes", "token = {:token}", "", 1, 0, dbx.Params{"token": token})
		if err == nil && len(records) > 0 {
			broadcastToDash(map[string]any{"action": "node_up", "data": records[0]})
		}
	}
}

func (m *WSManager) Unregister(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.conn, token)
	log.Printf("agent %s disconnected (%d remaining)", token[:8]+"...", len(m.conn))

	if m.hub == nil {
		return
	}
	records, err := m.hub.FindRecordsByFilter("nodes", "token = {:token}", "", 1, 0, dbx.Params{"token": token})
	if err == nil && len(records) > 0 {
		records[0].Set("status", "down")
		m.hub.Save(records[0])
		broadcastToDash(map[string]any{"action": "node_down", "data": records[0]})
	}
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

	ac.Conn.SetReadDeadline(time.Now().Add(pongWait))
	ac.Conn.SetPongHandler(func(string) error {
		ac.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

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
		case <-pingTicker.C:
			ac.mu.Lock()
			ac.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait))
			ac.mu.Unlock()
		default:
		}

		ac.Conn.SetReadDeadline(time.Now().Add(pongWait))
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
		case "heartbeat":
		}
	}
}

var (
	dashClients   = map[*websocket.Conn]bool{}
	dashClientsMu sync.RWMutex
)

func (h *Hub) handleDashboardWS(e *core.RequestEvent) error {
	conn, err := upgrader.Upgrade(e.Response, e.Request, nil)
	if err != nil {
		return nil
	}

	dashClientsMu.Lock()
	dashClients[conn] = true
	dashClientsMu.Unlock()

	defer func() {
		dashClientsMu.Lock()
		delete(dashClients, conn)
		dashClientsMu.Unlock()
		conn.Close()
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
	return nil
}

func broadcastToDash(msg any) {
	dashClientsMu.RLock()
	defer dashClientsMu.RUnlock()
	for conn := range dashClients {
		conn.WriteJSON(msg)
	}
}
