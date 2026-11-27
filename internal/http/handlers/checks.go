package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/services"
	"github.com/go-chi/chi/v5"
)

// StartCheck handles POST /v1/checks — '확인하기' {kind:"check", lectureId} and
// the diagnostics (P3): {kind:"pretest", lectureId|jobId} · {kind:"known_verify",
// conceptId} · {kind:"placement", chapterId}. The open session of the key is
// returned when there is one (201 = created). 204 = no pretest now.
func (h *LearnHandler) StartCheck(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Kind      string `json:"kind"`
		LectureID string `json:"lectureId"`
		JobID     string `json:"jobId"`
		ConceptID string `json:"conceptId"`
		ChapterID string `json:"chapterId"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	if body.Kind != "check" {
		h.startDiagnostic(w, r, userID, services.DiagnosticRequest{Kind: body.Kind, LectureID: body.LectureID,
			JobID: body.JobID, ConceptID: body.ConceptID, ChapterID: body.ChapterID})
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

// ErrorNoPlacement: the chapter has too few concepts left to diagnose.
const ErrorNoPlacement ErrorCode = "NO_PLACEMENT"

// ErrorNoSynthesis: not enough checked concepts for a synthesis, or one this week already.
const ErrorNoSynthesis ErrorCode = "NO_SYNTHESIS"

func (h *LearnHandler) startDiagnostic(w http.ResponseWriter, r *http.Request, userID string, req services.DiagnosticRequest) {
	d, created, err := h.Svc.StartDiagnostic(r.Context(), userID, req)
	switch {
	case errors.Is(err, services.ErrNoPlacement):
		writeError(w, r, http.StatusConflict, ErrorNoPlacement, err.Error())
	case errors.Is(err, services.ErrNoSynthesis):
		writeError(w, r, http.StatusConflict, ErrorNoSynthesis, err.Error())
	case err != nil:
		writeCheckError(w, r, "start diagnostic", err)
	case d == nil:
		w.WriteHeader(http.StatusNoContent)
	case created:
		writeJSON(w, http.StatusCreated, d)
	default:
		writeCheck(w, *d)
	}
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

// AnswerCheck handles POST /v1/checks/{id}/answers {text} | {choice} | {dontKnow:true} → 202 while the tutor replies.
func (h *LearnHandler) AnswerCheck(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Text string `json:"text"`
		// Confidence: sure | unsure | unknown before the first answer ("" = not chosen, v3 §3).
		Confidence string `json:"confidence"`
		// Choice / DontKnow: a diagnostic option (0-based) or '모르겠어요' (P3).
		Choice   *int `json:"choice"`
		DontKnow bool `json:"dontKnow"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	id := chi.URLParam(r, "id")
	if body.Choice != nil || body.DontKnow {
		d, err := h.Svc.AnswerChoice(r.Context(), userID, id, body.Choice, body.DontKnow)
		if err != nil {
			writeCheckError(w, r, "answer check", err)
			return
		}
		writeCheck(w, d)
		return
	}
	d, err := h.Svc.AnswerCheck(r.Context(), userID, id, body.Text, body.Confidence)
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

// SkipCheckItem handles POST /v1/checks/{id}/items/{itemId}/skip — '건너뛰기' on the
// current item before its first answer (409 when itemId is not the current item).
func (h *LearnHandler) SkipCheckItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	d, err := h.Svc.SkipCheck(r.Context(), userID, chi.URLParam(r, "id"), chi.URLParam(r, "itemId"))
	if err != nil {
		writeCheckError(w, r, "skip check item", err)
		return
	}
	writeCheck(w, d)
}

// RetryCheck handles POST /v1/checks/{id}/retry — rerun the failed generation or turn.
func (h *LearnHandler) RetryCheck(w http.ResponseWriter, r *http.Request) {
	h.checkAction(w, r, "retry check", h.Svc.RetryCheck)
}

// AbandonCheck handles POST /v1/checks/{id}/abandon (applied evidence stays).
func (h *LearnHandler) AbandonCheck(w http.ResponseWriter, r *http.Request) {
	h.checkAction(w, r, "abandon check", h.Svc.AbandonCheck)
}

// CancelCheck handles POST /v1/checks/{id}/cancel ([취소] while the items are
// being made): the generation stops and the session is abandoned; 409 once it is not preparing.
func (h *LearnHandler) CancelCheck(w http.ResponseWriter, r *http.Request) {
	h.checkAction(w, r, "cancel check", h.Svc.CancelCheck)
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

// Mastery handles GET /v1/mastery?conceptId= — per-objective level, next review, stale and grader —
// and GET /v1/mastery?conceptIds=a,b,c (max 300) — the concepts' levels only (the map's rings).
func (h *LearnHandler) Mastery(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	if q.Has("conceptIds") {
		ids, msg := parseConceptIDs(q.Get("conceptIds"))
		if msg != "" {
			writeError(w, r, http.StatusBadRequest, ErrorBadRequest, msg)
			return
		}
		list, err := h.Svc.MasteryBatch(r.Context(), userID, ids)
		if err != nil {
			writeCheckError(w, r, "mastery batch", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"concepts": list})
		return
	}
	conceptID := q.Get("conceptId")
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

// maxMasteryBatch bounds GET /v1/mastery?conceptIds= (a whole map).
const maxMasteryBatch = 300

// parseConceptIDs splits the comma list (trimmed, empty and repeated ids dropped).
func parseConceptIDs(raw string) ([]string, string) {
	seen := map[string]bool{}
	ids := []string{}
	for _, part := range strings.Split(raw, ",") {
		id := strings.TrimSpace(part)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, "conceptIds is required"
	}
	if len(ids) > maxMasteryBatch {
		return nil, fmt.Sprintf("conceptIds: at most %d", maxMasteryBatch)
	}
	return ids, ""
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
