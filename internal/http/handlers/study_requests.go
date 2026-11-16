package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/forgen-team/punisher/internal/curriculum"
	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/services"
	"github.com/forgen-team/punisher/internal/store"
	"github.com/go-chi/chi/v5"
)

const (
	maxQueryRunes     = 500
	maxSourceRefRunes = 500
	maxIdemKeyRunes   = 200
)

type StudyRequestsHandler struct {
	Store store.Store
	// Svc 가 있으면 별칭으로 바로 못 찾은 요청의 개념 해석을 즉시 백그라운드로 시작한다(스케줄러 30분 대기 없음).
	Svc *services.Service
}

type createStudyRequestBody struct {
	Source         string          `json:"source"`
	SourceRef      string          `json:"sourceRef"`
	Query          string          `json:"query"`
	Context        json.RawMessage `json:"context"`
	IdempotencyKey string          `json:"idempotencyKey"`
}

// Create handles POST /v1/study-requests. Idempotent per user by
// idempotencyKey (body or Idempotency-Key header; derived from
// source+sourceRef+query when absent). Concept resolution is an exact
// normalized alias/name/id match; otherwise status stays "unresolved".
// Responds 201 when created, 200 when an existing request is returned.
func (h *StudyRequestsHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body createStudyRequestBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	req, msg := validateStudyRequest(userID, body, r.Header.Get("Idempotency-Key"))
	if msg != "" {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, msg)
		return
	}

	resolved, err := h.Store.ResolveConcepts(r.Context(), userID, curriculum.NormalizeAlias(req.Query))
	if err != nil {
		log.Printf("resolve study request: %v", err)
		writeError(w, r, http.StatusInternalServerError, ErrorInternal, "failed to resolve concepts")
		return
	}
	if resolved == nil {
		resolved = []string{} // column is NOT NULL; nil would encode as NULL
	}
	req.ResolvedConceptIDs = resolved
	req.Status = "unresolved"
	if len(resolved) > 0 {
		req.Status = "resolved"
	}

	saved, created, err := h.Store.CreateStudyRequest(r.Context(), req)
	if err != nil {
		log.Printf("create study request: %v", err)
		writeError(w, r, http.StatusInternalServerError, ErrorInternal, "failed to save study request")
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	if saved.Status == "unresolved" && h.Svc != nil {
		h.Svc.ResolveRequestNow(userID, saved.ID)
		saved.Resolving = services.IsResolving(saved.ID)
	}
	writeJSON(w, status, map[string]any{"studyRequest": saved, "created": created})
}

// Resolve handles POST /v1/study-requests/{id}/resolve — "다시 시도". Starts concept resolution
// immediately regardless of past attempts. 202 when started, 200 when it is already resolved/running.
func (h *StudyRequestsHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	req, err := h.Store.GetStudyRequest(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, "get study request", err)
		return
	}
	if req.Status != "unresolved" || h.Svc == nil {
		req.Resolving = services.IsResolving(req.ID)
		writeJSON(w, http.StatusOK, map[string]any{"studyRequest": req})
		return
	}
	h.Svc.ResolveRequestNow(userID, req.ID)
	req.Resolving = true
	writeJSON(w, http.StatusAccepted, map[string]any{"studyRequest": req})
}

func validateStudyRequest(userID string, body createStudyRequestBody, headerKey string) (models.StudyRequest, string) {
	source := strings.TrimSpace(body.Source)
	if source == "" {
		source = learning.SourceManual
	}
	if !learning.IsValidSource(source) {
		return models.StudyRequest{}, "source must be rss, manual or guardian"
	}
	query := strings.TrimSpace(body.Query)
	if query == "" || utf8.RuneCountInString(query) > maxQueryRunes {
		return models.StudyRequest{}, "query is required (max 500 chars)"
	}
	sourceRef := strings.TrimSpace(body.SourceRef)
	if utf8.RuneCountInString(sourceRef) > maxSourceRefRunes {
		return models.StudyRequest{}, "sourceRef is too long (max 500 chars)"
	}

	key := strings.TrimSpace(body.IdempotencyKey)
	if key == "" {
		key = strings.TrimSpace(headerKey)
	}
	if utf8.RuneCountInString(key) > maxIdemKeyRunes {
		return models.StudyRequest{}, "idempotencyKey is too long (max 200 chars)"
	}
	if key == "" {
		key = learning.DeriveIdempotencyKey(source, sourceRef, curriculum.NormalizeAlias(query))
	}

	ctx := body.Context
	if len(ctx) == 0 || string(ctx) == "null" {
		ctx = json.RawMessage(`{}`)
	}
	return models.StudyRequest{
		UserID:         userID,
		Source:         source,
		SourceRef:      sourceRef,
		Query:          query,
		Context:        ctx,
		IdempotencyKey: key,
	}, ""
}
