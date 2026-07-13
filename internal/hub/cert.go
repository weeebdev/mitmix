package hub

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

func (h *Hub) handleCACert(e *core.RequestEvent) error {
	_, pem := h.ws.GetCACert()
	if pem == "" {
		return e.NotFoundError("no CA certificate registered", nil)
	}
	e.Response.Header().Set("Content-Type", "application/x-pem-file")
	e.Response.Header().Set("Content-Disposition", "attachment; filename=mitmproxy-ca-cert.pem")
	return e.String(http.StatusOK, pem)
}
