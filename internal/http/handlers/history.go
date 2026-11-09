package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/store"
)

// ListMyQuestions handles GET /v1/me/questions?conceptId=&limit=&cursor=.
func (h *LearnHandler) ListMyQuestions(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	limit, ok := readLimit(w, r, 20, 100)
	if !ok {
		return
	}
	q := r.URL.Query()
	items, next, err := h.Store.ListMyQuestions(r.Context(), userID, q.Get("conceptId"), limit, q.Get("cursor"))
	if err != nil {
		writeHistoryError(w, r, "list my questions", err)
		return
	}
	if items == nil {
		items = []models.MyQuestion{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"questions": items, "nextCursor": next})
}

// ListMyLectures handles GET /v1/me/lectures?status=completed|in_progress&limit=&cursor=.
func (h *LearnHandler) ListMyLectures(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	limit, ok := readLimit(w, r, 20, 100)
	if !ok {
		return
	}
	q := r.URL.Query()
	status := q.Get("status")
	if status == "" {
		status = "completed"
	}
	if status != "completed" && status != "in_progress" {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "status must be completed or in_progress")
		return
	}
	items, next, err := h.Store.ListMyLectures(r.Context(), userID, status, limit, q.Get("cursor"))
	if err != nil {
		writeHistoryError(w, r, "list my lectures", err)
		return
	}
	if items == nil {
		items = []models.MyLecture{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"lectures": items, "nextCursor": next})
}

// ConceptLectures handles GET /v1/concepts/{id}/lectures.
func (h *LearnHandler) ConceptLectures(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	items, err := h.Store.ListConceptLectures(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, "list concept lectures", err)
		return
	}
	if items == nil {
		items = []models.ConceptLecture{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"lectures": items})
}

func writeHistoryError(w http.ResponseWriter, r *http.Request, op string, err error) {
	if errors.Is(err, store.ErrInvalidCursor) {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "invalid cursor")
		return
	}
	writeServiceError(w, r, op, err)
}
