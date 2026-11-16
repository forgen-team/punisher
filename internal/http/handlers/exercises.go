package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wooo-jin/punisher/internal/services"
)

// ListExercises handles GET /v1/lectures/{id}/exercises — the caller's latest
// attempt per exercise/lab block and the lecture's '고칠 곳' (v3 §2, §6).
func (h *LearnHandler) ListExercises(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	lec, err := h.Store.GetLecture(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, "list exercises", err)
		return
	}
	attempts, err := h.Store.LatestExerciseAttempts(r.Context(), userID, lec.ID)
	if err != nil {
		writeServiceError(w, r, "list exercises", err)
		return
	}
	fixes, err := h.Svc.LectureFixes(r.Context(), userID, lec.ID)
	if err != nil {
		writeServiceError(w, r, "lecture fixes", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"attempts": attempts, "fixes": fixes})
}

// SubmitExercise handles POST /v1/lectures/{id}/exercises {anchor, contentHash, answer | skip}.
// 201 with the attempt; grading follows in the background (poll the list).
func (h *LearnHandler) SubmitExercise(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body services.ExerciseInput
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	if !body.Skip && !h.requireOwner(w, r, userID) {
		return
	}
	lec, err := h.Store.GetLecture(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, "submit exercise", err)
		return
	}
	a, err := h.Svc.SubmitExercise(r.Context(), userID, lec, body)
	switch {
	case errors.Is(err, services.ErrExerciseNotFound):
		writeError(w, r, http.StatusNotFound, ErrorNotFound, err.Error())
	case errors.Is(err, services.ErrExerciseStale):
		writeError(w, r, http.StatusConflict, ErrorConflict, err.Error())
	case errors.Is(err, services.ErrInvalidCheck):
		writeCheckError(w, r, "submit exercise", err)
	case err != nil:
		writeServiceError(w, r, "submit exercise", err)
	default:
		writeJSON(w, http.StatusCreated, map[string]any{"attempt": a})
	}
}

// DeleteExercise handles DELETE /v1/lectures/{id}/exercises/{attemptId} (e.g. a pasted lab output).
func (h *LearnHandler) DeleteExercise(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	if err := h.Store.DeleteExerciseAttempt(r.Context(), userID, chi.URLParam(r, "attemptId")); err != nil {
		writeServiceError(w, r, "delete exercise", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
