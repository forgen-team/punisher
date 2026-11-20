package handlers

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/services"
	"github.com/go-chi/chi/v5"
)

// Codes of the reading-thread 400 responses (learner-level ④).
const (
	ErrorInvalidThreadMode ErrorCode = "INVALID_THREAD_MODE"
	ErrorQuestionRequired  ErrorCode = "QUESTION_REQUIRED"
	ErrorInvalidQuote      ErrorCode = "INVALID_QUOTE"
)

// maxThreadQuestionRunes is the question limit of a thread turn (same as Ask).
const maxThreadQuestionRunes = 2000

type threadBody struct {
	Anchor   string `json:"anchor"`
	Quote    string `json:"quote"`
	Mode     string `json:"mode"`
	Question string `json:"question"`
}

type followupBody struct {
	Question string `json:"question"`
	Mode     string `json:"mode"`
}

// threadTurn validates the mode ("" = question) and question of a thread turn.
func threadTurn(mode, question string) (string, string, *settingsError) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = learning.ThreadQuestion
	}
	question = strings.TrimSpace(question)
	switch {
	case !learning.ValidThreadMode(mode):
		return mode, question, &settingsError{ErrorInvalidThreadMode, "mode must be easier|deeper|question"}
	case mode == learning.ThreadQuestion && question == "":
		return mode, question, &settingsError{ErrorQuestionRequired, "question is required for mode=question"}
	case utf8.RuneCountInString(question) > maxThreadQuestionRunes:
		return mode, question, &settingsError{ErrorBadRequest, "question max 2000 chars"}
	}
	return mode, question, nil
}

// validateThreadBody checks a new thread before any DB/LLM work.
func validateThreadBody(b threadBody) (services.ThreadRequest, *settingsError) {
	mode, question, invalid := threadTurn(b.Mode, b.Question)
	if invalid != nil {
		return services.ThreadRequest{}, invalid
	}
	anchor := strings.TrimSpace(b.Anchor)
	if utf8.RuneCountInString(anchor) > 200 {
		return services.ThreadRequest{}, &settingsError{ErrorBadRequest, "anchor max 200 chars"}
	}
	quote, err := services.NormalizeSelection(b.Quote)
	if err != nil {
		return services.ThreadRequest{}, &settingsError{ErrorInvalidQuote, "quote: " + err.Error()}
	}
	return services.ThreadRequest{Anchor: anchor, Quote: quote, Mode: mode, Question: question}, nil
}

// writeThreadAnswer answers 201 with the entry, or 502 with the stored failed entry.
func writeThreadAnswer(w http.ResponseWriter, q models.Question) {
	status := http.StatusCreated
	if q.Status == "failed" {
		status = http.StatusBadGateway
	}
	writeJSON(w, status, map[string]any{"question": q})
}

// CreateThread handles POST /v1/lectures/{id}/threads {anchor, quote, mode, question?}
// — the root of a reading thread, answered synchronously (short lane). Like
// Ask, a failed LLM call returns 502 with the stored failed entry.
func (h *LearnHandler) CreateThread(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body threadBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	req, invalid := validateThreadBody(body)
	if invalid != nil {
		writeError(w, r, http.StatusBadRequest, invalid.code, invalid.msg)
		return
	}
	if !h.requireOwner(w, r, userID) {
		return
	}
	lec, err := h.Store.GetLecture(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, "create thread", err)
		return
	}
	q, err := h.Svc.StartThread(r.Context(), userID, lec, req)
	if err != nil {
		writeServiceError(w, r, "create thread", err)
		return
	}
	writeThreadAnswer(w, q)
}

// CreateQuestionFollowup handles POST /v1/questions/{id}/followups {question?, mode?}
// — a follow-up in the thread of {id} (root or follow-up; parentId is the root).
func (h *LearnHandler) CreateQuestionFollowup(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body followupBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	mode, question, invalid := threadTurn(body.Mode, body.Question)
	if invalid != nil {
		writeError(w, r, http.StatusBadRequest, invalid.code, invalid.msg)
		return
	}
	if !h.requireOwner(w, r, userID) {
		return
	}
	q, err := h.Svc.Followup(r.Context(), userID, chi.URLParam(r, "id"), mode, question)
	if errors.Is(err, services.ErrNotThread) {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	if err != nil {
		writeServiceError(w, r, "question followup", err)
		return
	}
	writeThreadAnswer(w, q)
}

// DeepLecture handles POST /v1/questions/{id}/deep-lecture — an instant lecture
// from the thread, linked under its lecture. Same response as POST /v1/lectures/instant.
func (h *LearnHandler) DeepLecture(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	if !h.requireOwner(w, r, userID) {
		return
	}
	resp, err := h.Svc.DeepLecture(r.Context(), userID, chi.URLParam(r, "id"))
	if errors.Is(err, services.ErrNotThread) {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	if err != nil {
		// writeInquiryError: the same-concept fallback starts a question
		// explanation, whose busy/limit errors map like POST /v1/inquiries.
		writeInquiryError(w, r, "deep lecture", err)
		return
	}
	writeJSON(w, instantStatus(resp), resp)
}

// DismissLevelSuggestion handles POST /v1/me/level-suggestion/dismiss — hide
// the lower-level suggestion for a week (204).
func (h *LearnHandler) DismissLevelSuggestion(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	if err := h.Svc.DismissLevelSuggestion(r.Context(), userID); err != nil {
		writeServiceError(w, r, "dismiss level suggestion", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
