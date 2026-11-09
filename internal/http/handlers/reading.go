package handlers

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/wooo-jin/punisher/internal/services"
)

// Dismiss handles POST /v1/lectures/{id}/dismiss — hide from 이어보기 (soft).
func (h *LearnHandler) Dismiss(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.Store.DismissLecture(r.Context(), userID, id); err != nil {
		writeServiceError(w, r, "dismiss lecture", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lectureId": id, "dismissed": true})
}

// Restore handles DELETE /v1/lectures/{id}/dismiss — undo Dismiss.
func (h *LearnHandler) Restore(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.Store.RestoreLecture(r.Context(), userID, id); err != nil {
		writeServiceError(w, r, "restore lecture", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lectureId": id, "dismissed": false})
}

type explainBody struct {
	Anchor    string `json:"anchor"`
	Selection string `json:"selection"`
}

// validateExplainBody checks the request shape before any DB/LLM work.
func validateExplainBody(b explainBody) (explainBody, error) {
	b.Anchor = strings.TrimSpace(b.Anchor)
	if utf8.RuneCountInString(b.Anchor) > 200 {
		return b, errors.New("anchor max 200 chars")
	}
	sel, err := services.NormalizeSelection(b.Selection)
	if err != nil {
		return b, err
	}
	b.Selection = sel
	return b, nil
}

// Explain handles POST /v1/lectures/{id}/explain — "이 부분 설명" for a
// passage the user selected, answered synchronously (interactive lane).
func (h *LearnHandler) Explain(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body explainBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	body, err := validateExplainBody(body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	if !h.requireOwner(w, r, userID) {
		return
	}
	lec, err := h.Store.GetLecture(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, "explain", err)
		return
	}
	res, err := h.Svc.ExplainSelection(r.Context(), userID, lec, body.Anchor, body.Selection)
	if err != nil {
		writeServiceError(w, r, "explain", err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
