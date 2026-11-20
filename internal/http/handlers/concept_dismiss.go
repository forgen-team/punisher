package handlers

import (
	"errors"
	"net/http"

	"github.com/forgen-team/punisher/internal/store"
	"github.com/go-chi/chi/v5"
)

// DismissConcept handles POST /v1/concepts/{id}/dismiss — '필요 없음': the concept
// leaves today's recommendations, the frontier, library pre-generation and the
// review queue; it stays on the map (dimmed) and still locks its dependents.
func (h *LearnHandler) DismissConcept(w http.ResponseWriter, r *http.Request) {
	h.setConceptDismissed(w, r, true)
}

// RestoreConcept handles DELETE /v1/concepts/{id}/dismiss — '되돌리기'.
func (h *LearnHandler) RestoreConcept(w http.ResponseWriter, r *http.Request) {
	h.setConceptDismissed(w, r, false)
}

func (h *LearnHandler) setConceptDismissed(w http.ResponseWriter, r *http.Request, dismissed bool) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	var err error
	if dismissed {
		err = h.Store.DismissConcept(r.Context(), userID, id)
	} else {
		err = h.Store.RestoreConcept(r.Context(), userID, id)
	}
	if errors.Is(err, store.ErrConceptRetired) {
		writeError(w, r, http.StatusConflict, ErrorConflict, "concept is retired")
		return
	}
	if err != nil {
		writeServiceError(w, r, "dismiss concept", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"conceptId": id, "dismissed": dismissed})
}
