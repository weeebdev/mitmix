package hub

import (
	"crypto/rand"
	"log"
	"math/big"

	"github.com/pocketbase/pocketbase/core"
)

const base62Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func generateToken(length int) (string, error) {
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(62))
		if err != nil {
			return "", err
		}
		result[i] = base62Chars[n.Int64()]
	}
	return string(result), nil
}

func (h *Hub) handleListTokens(e *core.RequestEvent) error {
	records, err := h.FindRecordsByFilter("node_tokens", "1=1", "", 100, 0)
	if err != nil {
		log.Printf("tokens query error: %v", err)
		return e.InternalServerError("query failed", err)
	}
	return e.JSON(200, records)
}

type CreateTokenRequest struct {
	Label string `json:"label"`
}

func (h *Hub) handleCreateToken(e *core.RequestEvent) error {
	var req CreateTokenRequest
	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("invalid request", nil)
	}

	token, err := generateToken(32)
	if err != nil {
		log.Printf("token generation error: %v", err)
		return e.InternalServerError("token generation failed", nil)
	}

	col, err := h.FindCollectionByNameOrId("node_tokens")
	if err != nil {
		return e.InternalServerError("collection not found", nil)
	}

	rec := core.NewRecord(col)
	rec.Set("token", token)
	rec.Set("label", req.Label)

	if err := h.Save(rec); err != nil {
		log.Printf("create token error: %v", err)
		return e.InternalServerError("save failed", nil)
	}

	return e.JSON(201, rec)
}

func (h *Hub) handleDeleteToken(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	if id == "" {
		return e.BadRequestError("id is required", nil)
	}

	rec, err := h.FindRecordById("node_tokens", id)
	if err != nil {
		return e.NotFoundError("token not found", nil)
	}

	if err := h.Delete(rec); err != nil {
		log.Printf("delete token error: %v", err)
		return e.InternalServerError("delete failed", nil)
	}

	return e.JSON(200, map[string]string{"deleted": id})
}
