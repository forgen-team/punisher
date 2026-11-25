package handlers

import (
	"errors"
	"net/http"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/go-chi/chi/v5"
)

// '되돌리기' of the reversible learning actions (the UI's undo toast).

// Uncomplete handles DELETE /v1/lectures/{id}/complete — undo '완료'.
func (h *LearnHandler) Uncomplete(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	res, err := h.Store.UncompleteLecture(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, "uncomplete lecture", err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// UndoKnown handles DELETE /v1/concepts/{id}/known?previous=<state> — undo '이미 앎'
// back to the state the concept had before (not_started when omitted).
func (h *LearnHandler) UndoKnown(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	previous := r.URL.Query().Get("previous")
	if previous == "" {
		previous = learning.NotStarted
	}
	conceptID := chi.URLParam(r, "id")
	state, err := h.Store.UndoKnown(r.Context(), userID, conceptID, previous)
	if errors.Is(err, learning.ErrInvalidState) {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "previous must be not_started, in_progress or completed")
		return
	}
	if err != nil {
		writeServiceError(w, r, "undo known", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"conceptId": conceptID, "state": state})
}

// ResumeCheck handles POST /v1/checks/{id}/resume — undo '그만하기' shortly after.
func (h *LearnHandler) ResumeCheck(w http.ResponseWriter, r *http.Request) {
	h.checkAction(w, r, "resume check", h.Svc.ResumeCheck)
}
