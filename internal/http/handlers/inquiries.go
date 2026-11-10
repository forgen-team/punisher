package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/services"
	"github.com/wooo-jin/punisher/internal/store"
)

func writeInquiryError(w http.ResponseWriter, r *http.Request, op string, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidInquiry):
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
	case errors.Is(err, services.ErrInquiryBusy), errors.Is(err, services.ErrFollowupLimit), errors.Is(err, services.ErrInquiryNotReady):
		writeError(w, r, http.StatusConflict, ErrorConflict, err.Error())
	default:
		writeServiceError(w, r, op, err)
	}
}

// CreateInquiry handles POST /v1/inquiries {question} — '이 질문에 답하기'.
// Returns 202 with the pending inquiry; the explanation is written in the background.
func (h *LearnHandler) CreateInquiry(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Question string `json:"question"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	q, err := h.Svc.StartInquiry(r.Context(), userID, body.Question)
	if err != nil {
		writeInquiryError(w, r, "create inquiry", err)
		return
	}
	writeJSON(w, http.StatusAccepted, q)
}

// ListInquiries handles GET /v1/inquiries?open=1&concept=&limit= (no content).
func (h *LearnHandler) ListInquiries(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	qs := r.URL.Query()
	limit, _ := strconv.Atoi(qs.Get("limit"))
	list, err := h.Store.ListInquiries(r.Context(), userID, store.InquiryFilter{
		Open: qs.Get("open") == "1", ConceptID: qs.Get("concept"), Limit: limit,
	})
	if err != nil {
		writeServiceError(w, r, "list inquiries", err)
		return
	}
	if list == nil {
		list = []models.Inquiry{}
	}
	for i := range list {
		list[i] = services.PublicInquiry(list[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{"inquiries": list})
}

// GetInquiry handles GET /v1/inquiries/{id} (with content and concepts).
func (h *LearnHandler) GetInquiry(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	q, err := h.Svc.GetInquiry(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, "get inquiry", err)
		return
	}
	writeJSON(w, http.StatusOK, q)
}

// RegenerateInquiry handles POST /v1/inquiries/{id}/regenerate ('다시 만들기').
func (h *LearnHandler) RegenerateInquiry(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	q, err := h.Svc.RegenerateInquiry(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeInquiryError(w, r, "regenerate inquiry", err)
		return
	}
	writeJSON(w, http.StatusAccepted, services.PublicInquiry(q))
}

// MarkInquiryRead handles POST (read) / DELETE (restore) /v1/inquiries/{id}/read.
func (h *LearnHandler) MarkInquiryRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	if err := h.Store.SetInquiryRead(r.Context(), userID, chi.URLParam(r, "id"), r.Method == http.MethodPost); err != nil {
		writeServiceError(w, r, "read inquiry", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CreateFollowup handles POST /v1/inquiries/{id}/followups {question, quote} — 이어서 묻기.
func (h *LearnHandler) CreateFollowup(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Question string `json:"question"`
		Quote    string `json:"quote"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	f, err := h.Svc.StartFollowup(r.Context(), userID, chi.URLParam(r, "id"), body.Question, body.Quote)
	if err != nil {
		writeInquiryError(w, r, "create followup", err)
		return
	}
	writeJSON(w, http.StatusAccepted, f)
}

// RetryFollowup handles POST /v1/inquiries/{id}/followups/{followupId}/retry.
func (h *LearnHandler) RetryFollowup(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	f, err := h.Svc.RetryFollowup(r.Context(), userID, chi.URLParam(r, "id"), chi.URLParam(r, "followupId"))
	if err != nil {
		writeInquiryError(w, r, "retry followup", err)
		return
	}
	writeJSON(w, http.StatusAccepted, f)
}

// ListCompleted handles GET /v1/me/completed?area=&q=&limit=&cursor= — 완료한 강의와 질문 스레드(최근 순, 서버에서 거른다).
func (h *LearnHandler) ListCompleted(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	limit, ok := readLimit(w, r, 20, 100)
	if !ok {
		return
	}
	qs := r.URL.Query()
	items, next, err := h.Store.ListCompleted(r.Context(), userID, store.CompletedFilter{Area: qs.Get("area"), Query: qs.Get("q"), Limit: limit}, qs.Get("cursor"))
	if err != nil {
		writeHistoryError(w, r, "list completed", err)
		return
	}
	if items == nil {
		items = []models.CompletedItem{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "nextCursor": next})
}

// CompletedAreas handles GET /v1/me/completed/areas?q= — 분야별 개수.
func (h *LearnHandler) CompletedAreas(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	areas, err := h.Store.CompletedAreas(r.Context(), userID, r.URL.Query().Get("q"))
	if err != nil {
		writeHistoryError(w, r, "completed areas", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"areas": areas})
}
