package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/llm"
	"github.com/wooo-jin/punisher/internal/services"
	"github.com/wooo-jin/punisher/internal/store"
)

// LearnHandler serves lectures, today, Q&A, notes and goals.
type LearnHandler struct {
	Store store.Store
	Svc   *services.Service
}

const (
	ErrorForbidden      ErrorCode = "FORBIDDEN"
	ErrorBudgetExceeded ErrorCode = "BUDGET_EXCEEDED"
	ErrorLLMFailed      ErrorCode = "LLM_FAILED"
	// ErrorGenUnavailable: local claude CLI missing or not logged in.
	ErrorGenUnavailable ErrorCode = "GENERATION_UNAVAILABLE"
	ErrorLLMRateLimited ErrorCode = "LLM_RATE_LIMITED"
)

// writeServiceError maps service/store errors to HTTP responses.
func writeServiceError(w http.ResponseWriter, r *http.Request, op string, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidSelection):
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
	case errors.Is(err, store.ErrNotFound):
		writeError(w, r, http.StatusNotFound, ErrorNotFound, "not found")
	case errors.Is(err, services.ErrNotOwner):
		writeError(w, r, http.StatusForbidden, ErrorForbidden, err.Error())
	case errors.Is(err, services.ErrBudgetExceeded):
		writeError(w, r, http.StatusTooManyRequests, ErrorBudgetExceeded, err.Error())
	case errors.Is(err, services.ErrTargetBusy), errors.Is(err, store.ErrJobActive):
		writeError(w, r, http.StatusConflict, ErrorConflict, err.Error())
	case errors.Is(err, llm.ErrGenerationUnavailable):
		log.Printf("%s: %v", op, err)
		writeError(w, r, http.StatusServiceUnavailable, ErrorGenUnavailable, services.DescribeLLMError(err))
	case errors.Is(err, llm.ErrRateLimited):
		writeError(w, r, http.StatusTooManyRequests, ErrorLLMRateLimited, services.DescribeLLMError(err))
	case isLLMError(err):
		log.Printf("%s: %v", op, err)
		writeError(w, r, http.StatusBadGateway, ErrorLLMFailed, services.DescribeLLMError(err))
	default:
		log.Printf("%s: %v", op, err)
		writeError(w, r, http.StatusInternalServerError, ErrorInternal, op+" failed")
	}
}

func isLLMError(err error) bool {
	var llmErr *llm.LLMError
	return errors.As(err, &llmErr) || errors.Is(err, services.ErrUnusableOutput)
}

// requireOwner blocks LLM features for non-owner accounts.
func (h *LearnHandler) requireOwner(w http.ResponseWriter, r *http.Request, userID string) bool {
	if !h.Svc.AllowUser(userID) {
		writeError(w, r, http.StatusForbidden, ErrorForbidden, services.ErrNotOwner.Error())
		return false
	}
	return true
}

// Today handles GET /v1/today?limit=.
func (h *LearnHandler) Today(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	limit, ok := readLimit(w, r, 5, 20)
	if !ok {
		return
	}
	res, err := h.Svc.Today(r.Context(), userID, limit)
	if err != nil {
		writeServiceError(w, r, "today", err)
		return
	}
	payload := map[string]any{"items": res.Items, "exhausted": res.Exhausted, "review": res.Review, "recheck": res.Recheck,
		"session": res.Session}
	if res.NextChapter != nil {
		payload["nextChapter"] = res.NextChapter
	}
	if res.Exhausted {
		payload["message"] = "추천할 다음 강의가 없습니다. 새 목표를 선택해 주세요."
	}
	writeJSON(w, http.StatusOK, payload)
}

// GetLecture handles GET /v1/lectures/{id}.
func (h *LearnHandler) GetLecture(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	lec, err := h.Store.GetLecture(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, "get lecture", err)
		return
	}
	// 섹션별 시간은 모델이 쓴 minutes 대신 서버 계산값(v2.2 §5)을 함께 준다. 전체 시간은 그 섹션 분의 합
	// (저장된 값이 예전 계산이어도 머리말과 섹션 합이 어긋나지 않게).
	var total int
	lec.Content, total = learning.WithSectionMinutes(lec.Content)
	if total > 0 {
		lec.EstimatedMinutes = total
	}
	writeJSON(w, http.StatusOK, lec)
}

type progressBody struct {
	Anchor  string `json:"anchor"`
	Percent int    `json:"percent"`
}

// Progress handles PUT /v1/lectures/{id}/progress.
func (h *LearnHandler) Progress(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body progressBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	if body.Percent < 0 || body.Percent > 100 || utf8.RuneCountInString(body.Anchor) > 200 {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "percent must be 0..100, anchor max 200 chars")
		return
	}
	if err := h.Store.UpdateProgress(r.Context(), userID, chi.URLParam(r, "id"), strings.TrimSpace(body.Anchor), body.Percent); err != nil {
		writeServiceError(w, r, "update progress", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"anchor": body.Anchor, "percent": body.Percent})
}

// Complete handles POST /v1/lectures/{id}/complete.
func (h *LearnHandler) Complete(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	res, err := h.Store.CompleteLecture(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, "complete lecture", err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// MarkKnown handles POST /v1/concepts/{id}/known [{queueVerify}].
func (h *LearnHandler) MarkKnown(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		// QueueVerify: bulk '이미 앎' — the objectives wait in the verification queue (P3).
		QueueVerify bool `json:"queueVerify"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	conceptID := chi.URLParam(r, "id")
	state, err := h.Svc.MarkKnown(r.Context(), userID, conceptID, body.QueueVerify)
	if errors.Is(err, store.ErrConceptRetired) {
		writeError(w, r, http.StatusConflict, ErrorConflict, "concept is retired")
		return
	}
	if err != nil {
		writeServiceError(w, r, "mark known", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"conceptId": conceptID, "state": state})
}

var feedbackValues = map[string]bool{"understood": true, "insufficient": true, "too_easy": true, "too_hard": true, "want_more": true}

type feedbackBody struct {
	Feedback string `json:"feedback"`
	Anchor   string `json:"anchor"`
	Comment  string `json:"comment"`
}

// Feedback handles POST /v1/lectures/{id}/feedback. "insufficient" also
// creates an alternative-explanation request answered in the background.
func (h *LearnHandler) Feedback(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body feedbackBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	if !feedbackValues[body.Feedback] || utf8.RuneCountInString(body.Comment) > 2000 {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "feedback must be understood|insufficient|too_easy|too_hard|want_more (comment max 2000)")
		return
	}
	lec, err := h.Store.GetLecture(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, "feedback", err)
		return
	}
	resp := map[string]any{"feedback": body.Feedback}
	altID := ""
	if body.Feedback == "insufficient" && h.Svc.AllowUser(userID) {
		q, err := h.Svc.RequestAltExplanation(r.Context(), userID, lec, body.Anchor, body.Comment)
		if err != nil {
			// Feedback is still recorded; the client sees why no re-explanation was started.
			resp["altRequestError"] = err.Error()
		} else {
			altID = q.ID
			resp["altRequestId"] = q.ID
			resp["altRequestStatus"] = q.Status
		}
	}
	id, err := h.Store.InsertFeedback(r.Context(), userID, lec.ID, body.Feedback, body.Anchor, body.Comment, altID)
	if err != nil {
		writeServiceError(w, r, "insert feedback", err)
		return
	}
	resp["id"] = id
	writeJSON(w, http.StatusCreated, resp)
}

func readLimit(w http.ResponseWriter, r *http.Request, def, max int) (int, bool) {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return def, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > max {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "limit out of range")
		return 0, false
	}
	return n, true
}
