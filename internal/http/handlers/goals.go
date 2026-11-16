package handlers

import (
	appMiddleware "github.com/forgen-team/punisher/internal/middleware"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/services"
	"github.com/go-chi/chi/v5"
)

var goalStatuses = map[string]bool{"active": true, "achieved": true, "archived": true}

type goalBody struct {
	Title            *string  `json:"title"`
	TargetConceptIDs []string `json:"targetConceptIds"`
	Status           *string  `json:"status"`
}

// validTargets checks that every target concept exists (non-retired).
func (h *LearnHandler) validTargets(r *http.Request, ids []string) (bool, error) {
	if len(ids) == 0 || len(ids) > 10 {
		return false, nil
	}
	userID, err := appMiddleware.GetUserID(r.Context())
	if err != nil {
		return false, err
	}
	known, err := h.Store.KnownConceptIDs(r.Context(), userID)
	if err != nil {
		return false, err
	}
	for _, id := range ids {
		if !known[id] {
			return false, nil
		}
	}
	return true, nil
}

// ListGoals handles GET /v1/goals?status=.
func (h *LearnHandler) ListGoals(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	status := r.URL.Query().Get("status")
	if status != "" && !goalStatuses[status] {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "status must be active|achieved|archived")
		return
	}
	goals, err := h.Store.ListGoals(r.Context(), userID, status)
	if err != nil {
		writeServiceError(w, r, "list goals", err)
		return
	}
	if goals == nil {
		goals = []models.Goal{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"goals": goals})
}

// CreateGoal handles POST /v1/goals {title, targetConceptIds}.
func (h *LearnHandler) CreateGoal(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body goalBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	title := str(body.Title)
	valid, err := h.validTargets(r, body.TargetConceptIDs)
	if err != nil {
		writeServiceError(w, r, "create goal", err)
		return
	}
	if title == "" || utf8.RuneCountInString(title) > 200 || !valid {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "title (max 200) and 1..10 existing targetConceptIds are required")
		return
	}
	g, err := h.Store.CreateGoal(r.Context(), userID, title, body.TargetConceptIDs)
	if err != nil {
		writeServiceError(w, r, "create goal", err)
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

// UpdateGoal handles PATCH /v1/goals/{id}.
func (h *LearnHandler) UpdateGoal(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body goalBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	if body.Status != nil && !goalStatuses[*body.Status] {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "status must be active|achieved|archived")
		return
	}
	if body.Title != nil && (strings.TrimSpace(*body.Title) == "" || utf8.RuneCountInString(*body.Title) > 200) {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "title must be 1..200 chars")
		return
	}
	if body.TargetConceptIDs != nil {
		valid, err := h.validTargets(r, body.TargetConceptIDs)
		if err != nil {
			writeServiceError(w, r, "update goal", err)
			return
		}
		if !valid {
			writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "1..10 existing targetConceptIds required")
			return
		}
	}
	g, err := h.Store.UpdateGoal(r.Context(), userID, chi.URLParam(r, "id"), body.Title, body.TargetConceptIDs, body.Status)
	if err != nil {
		writeServiceError(w, r, "update goal", err)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

// DeleteGoal handles DELETE /v1/goals/{id}.
func (h *LearnHandler) DeleteGoal(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	if err := h.Store.DeleteGoal(r.Context(), userID, chi.URLParam(r, "id")); err != nil {
		writeServiceError(w, r, "delete goal", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GoalPath handles GET /v1/goals/{id}/path — prerequisite chain per target
// with user states and the next concept to study.
func (h *LearnHandler) GoalPath(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	g, err := h.Store.GetGoal(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, "goal path", err)
		return
	}
	type targetPath struct {
		TargetConceptID   string            `json:"targetConceptId"`
		StartingConceptID string            `json:"startingConceptId"`
		Completed         bool              `json:"completed"`
		Steps             []models.PathStep `json:"steps"`
	}
	paths := make([]targetPath, 0, len(g.TargetConceptIDs))
	for _, target := range g.TargetConceptIDs {
		steps, start, err := h.Svc.GoalPath(r.Context(), userID, target)
		if err != nil {
			writeServiceError(w, r, "goal path", err)
			return
		}
		paths = append(paths, targetPath{TargetConceptID: target, StartingConceptID: start, Completed: start == "" && len(steps) > 0, Steps: steps})
	}
	writeJSON(w, http.StatusOK, map[string]any{"goal": g, "paths": paths})
}

// ListStudyRequests handles GET /v1/study-requests?status=&limit= (RSS status view).
func (h *LearnHandler) ListStudyRequests(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	limit, ok := readLimit(w, r, 50, 200)
	if !ok {
		return
	}
	list, err := h.Store.ListStudyRequests(r.Context(), userID, r.URL.Query().Get("status"), limit)
	if err != nil {
		writeServiceError(w, r, "list study requests", err)
		return
	}
	if list == nil {
		list = []models.StudyRequest{}
	}
	for i := range list {
		list[i].Resolving = services.IsResolving(list[i].ID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"studyRequests": list})
}

// GetStudyRequest handles GET /v1/study-requests/{id}. `learned` is true when
// every resolved concept is completed/known ("이미 학습함").
func (h *LearnHandler) GetStudyRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	req, err := h.Store.GetStudyRequest(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, "get study request", err)
		return
	}
	req.Resolving = services.IsResolving(req.ID)
	writeJSON(w, http.StatusOK, map[string]any{"studyRequest": req})
}
