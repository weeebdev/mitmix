package hub

import (
	"bufio"
	"log"
	"net"
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func (h *Hub) handleAgentConnect(e *core.RequestEvent) error {
	token := e.Request.Header.Get("X-Token")
	if token == "" {
		return e.BadRequestError("missing X-Token header", nil)
	}

	_, err := h.FindFirstRecordByFilter("node_tokens", "token = {:token}", dbx.Params{"token": token})
	if err != nil {
		return e.UnauthorizedError("invalid token", nil)
	}

	hj, ok := e.Response.(http.Hijacker)
	if !ok {
		return e.InternalServerError("server does not support hijacking", nil)
	}
	conn, bufrw, err := hj.Hijack()
	if err != nil {
		return e.InternalServerError("hijack failed", nil)
	}
	go h.handleWS(conn, bufrw, token)
	return nil
}

func (h *Hub) handleWS(conn net.Conn, bufrw *bufio.ReadWriter, token string) {
	defer conn.Close()
	log.Printf("agent connected (token=%.8s...)", token)
}
