package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/services"
)

func writeCheckError(w http.ResponseWriter, r *http.Request, op string, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidCheck):
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
	case errors.Is(err, services.ErrCheckBusy), errors.Is(err, services.ErrCheckState):
		writeError(w, r, http.StatusConflict, ErrorConflict, err.Error())
	default:
		writeServiceError(w, r, op, err)
	}
}

// writeCheck answers 202 while a tutor turn or the item generation runs (poll GET), else 200.
func writeCheck(w http.ResponseWriter, d models.CheckDetail) {
	status := http.StatusOK
	if d.Pending {
		status = http.StatusAccepted
	}
	writeJSON(w, status, d)
}

// StartCheck handles POST /v1/checks {kind:"check", lectureId} — '확인하기'.
// Returns the open session of the lecture when there is one (201 = created).
func (h *LearnHandler) StartCheck(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Kind      string `json:"kind"`
		LectureID string `json:"lectureId"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	d, created, err := h.Svc.StartCheck(r.Context(), userID, body.Kind, body.LectureID)
	if err != nil {
		writeCheckError(w, r, "start check", err)
		return
	}
	if created {
		writeJSON(w, http.StatusCreated, d)
		return
	}
	writeCheck(w, d)
}

// GetCheck handles GET /v1/checks/{id} — session, items, turns and the objectives' mastery.
func (h *LearnHandler) GetCheck(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	d, err := h.Svc.GetCheck(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeCheckError(w, r, "get check", err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// AnswerCheck handles POST /v1/checks/{id}/answers {text} → 202 (the tutor replies in the background).
func (h *LearnHandler) AnswerCheck(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Text string `json:"text"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	d, err := h.Svc.AnswerCheck(r.Context(), userID, chi.URLParam(r, "id"), body.Text)
	if err != nil {
		writeCheckError(w, r, "answer check", err)
		return
	}
	writeCheck(w, d)
}

// RevealCheck handles POST /v1/checks/{id}/reveal — '그냥 알려줘' on the current item.
func (h *LearnHandler) RevealCheck(w http.ResponseWriter, r *http.Request) {
	h.checkAction(w, r, "reveal check", h.Svc.RevealCheck)
}

// RetryCheck handles POST /v1/checks/{id}/retry — rerun the failed generation or turn.
func (h *LearnHandler) RetryCheck(w http.ResponseWriter, r *http.Request) {
	h.checkAction(w, r, "retry check", h.Svc.RetryCheck)
}

// AbandonCheck handles POST /v1/checks/{id}/abandon (applied evidence stays).
func (h *LearnHandler) AbandonCheck(w http.ResponseWriter, r *http.Request) {
	h.checkAction(w, r, "abandon check", h.Svc.AbandonCheck)
}

func (h *LearnHandler) checkAction(w http.ResponseWriter, r *http.Request, op string,
	run func(ctx context.Context, userID, id string) (models.CheckDetail, error)) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	d, err := run(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeCheckError(w, r, op, err)
		return
	}
	writeCheck(w, d)
}

// SelfGradeCheck handles POST /v1/checks/{id}/self-grade {verdict} (self-graded sessions, P2).
func (h *LearnHandler) SelfGradeCheck(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Verdict string `json:"verdict"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	d, err := h.Svc.SelfGradeCheck(r.Context(), userID, chi.URLParam(r, "id"), body.Verdict)
	if err != nil {
		writeCheckError(w, r, "self-grade check", err)
		return
	}
	writeCheck(w, d)
}

// DisputeCheck handles POST /v1/checks/{id}/turns/{turnId}/dispute [{reason}] — '내 답이 맞아요' → 202.
func (h *LearnHandler) DisputeCheck(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	d, err := h.Svc.DisputeCheck(r.Context(), userID, chi.URLParam(r, "id"), chi.URLParam(r, "turnId"), body.Reason)
	if err != nil {
		writeCheckError(w, r, "dispute check", err)
		return
	}
	writeCheck(w, d)
}

// Relearn handles POST /v1/objectives/{id}/relearn — '다시 배우기' (returns the concept's mastery).
func (h *LearnHandler) Relearn(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	m, err := h.Svc.Relearn(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeCheckError(w, r, "relearn", err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// Mastery handles GET /v1/mastery?conceptId= — per-objective level, next review, stale and grader.
func (h *LearnHandler) Mastery(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	conceptID := r.URL.Query().Get("conceptId")
	if conceptID == "" {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "conceptId is required")
		return
	}
	m, err := h.Svc.Mastery(r.Context(), userID, conceptID)
	if err != nil {
		writeCheckError(w, r, "mastery", err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// decodeOptionalJSON is decodeJSON that accepts an empty body.
func decodeOptionalJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	return nil
}
