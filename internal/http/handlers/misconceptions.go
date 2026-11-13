package handlers

import (
	"net/http"
)

// AcknowledgeMisconception handles POST /v1/misconceptions {turnId} — '맞아요,
// 그렇게 알았어요' on a check result (201 created, 200 already acknowledged,
// 404 when the turn is not the user's or has no misconception).
func (h *LearnHandler) AcknowledgeMisconception(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		TurnID string `json:"turnId"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	m, created, err := h.Svc.AcknowledgeMisconception(r.Context(), userID, body.TurnID)
	if err != nil {
		writeCheckError(w, r, "acknowledge misconception", err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, map[string]any{"misconception": m})
}

// ListMisconceptions handles GET /v1/misconceptions?status=open|resolved — latest first.
func (h *LearnHandler) ListMisconceptions(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	list, err := h.Svc.ListMisconceptions(r.Context(), userID, r.URL.Query().Get("status"))
	if err != nil {
		writeCheckError(w, r, "list misconceptions", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"misconceptions": list})
}
