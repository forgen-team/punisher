package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/store"
	"github.com/go-chi/chi/v5"
)

type ProgressHandler struct {
	Store store.Store
}

type setStateRequest struct {
	State string `json:"state"`
}

// SetConceptState handles PUT /v1/me/concepts/{id}/state.
// Body: {"state": "in_progress" | "completed" | "known" | "not_started"}.
// completed/known are only set through this explicit call.
func (h *ProgressHandler) SetConceptState(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body setStateRequest
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}

	conceptID := chi.URLParam(r, "id")
	state, err := h.Store.SetConceptState(r.Context(), userID, conceptID, body.State)
	switch {
	case errors.Is(err, learning.ErrInvalidState):
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "state must be in_progress, completed, known or not_started")
		return
	case errors.Is(err, store.ErrNotFound):
		writeError(w, r, http.StatusNotFound, ErrorNotFound, "concept not found")
		return
	case errors.Is(err, store.ErrConceptRetired):
		writeError(w, r, http.StatusConflict, ErrorConflict, "concept is retired")
		return
	case err != nil:
		log.Printf("set concept state: %v", err)
		writeError(w, r, http.StatusInternalServerError, ErrorInternal, "failed to update state")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"conceptId": conceptID, "state": state})
}
