package handlers

import (
	"errors"
	"net/http"

	"github.com/forgen-team/punisher/internal/check"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/services"
)

// ErrorNoReviews: nothing is due today (or review time is 0 minutes).
const ErrorNoReviews ErrorCode = "NO_REVIEWS"

// StartReview handles POST /v1/reviews — today's review session (201 = created,
// 200 = the open one). Answer / self-grade / reveal / abandon use /v1/checks/{id}/….
func (h *LearnHandler) StartReview(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	d, created, err := h.Svc.StartReview(r.Context(), userID)
	if errors.Is(err, services.ErrNoReviews) {
		writeError(w, r, http.StatusConflict, ErrorNoReviews, err.Error())
		return
	}
	if err != nil {
		writeCheckError(w, r, "start review", err)
		return
	}
	if created {
		writeJSON(w, http.StatusCreated, d)
		return
	}
	writeCheck(w, d)
}

// ReviewSummary handles GET /v1/reviews/summary — due · overdue · queued, estimate, today's session.
func (h *LearnHandler) ReviewSummary(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	sum, err := h.Svc.ReviewSummary(r.Context(), userID)
	if err != nil {
		writeServiceError(w, r, "review summary", err)
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

// CheckHistory handles GET /v1/checks/history?kind=check|review|known_verify&limit=&cursor= — sessions, newest first.
func (h *LearnHandler) CheckHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	limit, ok := readLimit(w, r, 20, 100)
	if !ok {
		return
	}
	q := r.URL.Query()
	kind := q.Get("kind")
	if kind != "" && kind != check.PurposeCheck && kind != check.PurposeReview && kind != check.PurposeKnownVerify {
		writeError(w, r, http.StatusBadRequest, services.CodeInvalidKind, "kind must be check|review|known_verify")
		return
	}
	items, next, err := h.Store.ListCheckHistory(r.Context(), userID, kind, limit, q.Get("cursor"))
	if err != nil {
		writeHistoryError(w, r, "check history", err)
		return
	}
	if items == nil {
		items = []models.CheckHistoryItem{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": items, "nextCursor": next})
}

// MasteryMetrics handles GET /v1/mastery/metrics — first-attempt review accuracy
// on unseen items by interval and grader, last 4 weeks (rate null = no attempt).
func (h *LearnHandler) MasteryMetrics(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	m, err := h.Svc.Metrics(r.Context(), userID)
	if err != nil {
		writeServiceError(w, r, "mastery metrics", err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}
