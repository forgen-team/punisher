package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/wooo-jin/punisher/internal/curriculum"
	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/store"
)

type ConceptsHandler struct {
	Store store.Store
}

var conceptStatuses = map[string]bool{"active": true, "proposed": true, "retired": true, "all": true}

// List handles GET /v1/concepts?area=&status= (status defaults to active; "all" disables it).
func (h *ConceptsHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	area, ok := readArea(w, r)
	if !ok {
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = "active"
	}
	if !conceptStatuses[status] {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "status must be active, proposed, retired or all")
		return
	}
	if status == "all" {
		status = ""
	}

	concepts, err := h.Store.ListConcepts(r.Context(), userID, store.ConceptFilter{Area: area, Status: status})
	if err != nil {
		log.Printf("list concepts: %v", err)
		writeError(w, r, http.StatusInternalServerError, ErrorInternal, "failed to list concepts")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"concepts": concepts})
}

// Detail handles GET /v1/concepts/{id}.
func (h *ConceptsHandler) Detail(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	detail, err := h.Store.GetConceptDetail(r.Context(), userID, chi.URLParam(r, "id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, r, http.StatusNotFound, ErrorNotFound, "concept not found")
		return
	}
	if err != nil {
		log.Printf("concept detail: %v", err)
		writeError(w, r, http.StatusInternalServerError, ErrorInternal, "failed to load concept")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// Graph handles GET /v1/graph?area= — the knowledge-map payload. Lock/frontier
// flags are computed on the full active graph, then filtered by area.
func (h *ConceptsHandler) Graph(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	area, ok := readArea(w, r)
	if !ok {
		return
	}
	data, err := h.Store.LoadGraph(r.Context(), userID)
	if err == nil {
		err = h.Store.LoadMapExtras(r.Context(), userID, data)
	}
	if err != nil {
		log.Printf("load graph: %v", err)
		writeError(w, r, http.StatusInternalServerError, ErrorInternal, "failed to load graph")
		return
	}
	writeJSON(w, http.StatusOK, buildGraph(data, area))
}

// Frontier handles GET /v1/me/frontier?limit= — concepts whose prerequisites
// are all completed/known but which are not themselves completed/known.
func (h *ConceptsHandler) Frontier(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 200 {
			writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "limit must be 1..200")
			return
		}
		limit = n
	}
	data, err := h.Store.LoadGraph(r.Context(), userID)
	if err != nil {
		log.Printf("load graph for frontier: %v", err)
		writeError(w, r, http.StatusInternalServerError, ErrorInternal, "failed to compute frontier")
		return
	}

	analysis := analyze(data)
	byID := make(map[string]models.Concept, len(data.Concepts))
	for _, c := range data.Concepts {
		byID[c.ID] = c
	}
	items := make([]models.FrontierItem, 0, limit)
	for _, f := range analysis.Frontier {
		if len(items) == limit {
			break
		}
		c := byID[f.ID]
		items = append(items, models.FrontierItem{
			ID: c.ID, Name: c.Name, Area: c.Area, Depth: c.Depth, Summary: c.Summary,
			State: f.State, Unlocks: f.Unlocks,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"total": len(analysis.Frontier),
	})
}

func readArea(w http.ResponseWriter, r *http.Request) (string, bool) {
	area := strings.TrimSpace(r.URL.Query().Get("area"))
	if area != "" && !curriculum.IsValidArea(area) {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "unknown area")
		return "", false
	}
	return area, true
}

func analyze(data *models.GraphData) learning.Analysis {
	nodes := make([]learning.Node, 0, len(data.Concepts))
	for _, c := range data.Concepts {
		nodes = append(nodes, learning.Node{ID: c.ID, Depth: c.Depth})
	}
	var prereqs []learning.PrereqEdge
	for _, e := range data.Edges {
		if e.Kind == curriculum.EdgePrerequisite {
			prereqs = append(prereqs, learning.PrereqEdge{From: e.From, To: e.To})
		}
	}
	return learning.Analyze(nodes, prereqs, data.States)
}

func buildGraph(data *models.GraphData, area string) models.Graph {
	analysis := analyze(data)
	graph := models.Graph{
		CurriculumVersion: data.CurriculumVersion,
		Nodes:             []models.GraphNode{},
		Edges:             []models.Edge{},
		PendingRequests:   []models.MapRequest{},
	}
	requestsByConcept := map[string][]models.MapRequest{}
	for _, req := range data.Requests {
		if len(req.ConceptIDs) == 0 {
			graph.PendingRequests = append(graph.PendingRequests, req)
			continue
		}
		for _, id := range req.ConceptIDs {
			requestsByConcept[id] = append(requestsByConcept[id], req)
		}
	}

	included := make(map[string]bool, len(data.Concepts))
	coverage := map[string]*models.AreaCoverage{}
	for _, c := range data.Concepts {
		a := analysis.Nodes[c.ID]
		done := learning.IsDone(a.State)
		var requests []models.MapRequest
		if !done {
			requests = requestsByConcept[c.ID]
		}
		countCoverage(coverage, c, a.State, len(requests) > 0, !done && data.Dismissed[c.ID])

		if area != "" && c.Area != area {
			continue
		}
		included[c.ID] = true
		graph.Nodes = append(graph.Nodes, models.GraphNode{
			ID: c.ID, Name: c.Name, Area: c.Area, Depth: c.Depth, Status: c.Status,
			State: a.State, Locked: a.Locked, Frontier: a.Frontier, Unlocks: a.Unlocks, Requests: requests,
		})
	}
	for _, e := range data.Edges {
		if included[e.From] && included[e.To] {
			graph.Edges = append(graph.Edges, e)
		}
	}
	for _, a := range curriculum.Areas {
		if cov := coverage[a]; cov != nil && (area == "" || area == a) {
			graph.Coverage = append(graph.Coverage, *cov)
		}
	}
	return graph
}

// countCoverage adds one concept to its area's summary. Total/completed/known
// count curriculum (active) concepts only; requested also counts proposed ones.
func countCoverage(coverage map[string]*models.AreaCoverage, c models.Concept, state string, requested, dismissed bool) {
	cov := coverage[c.Area]
	if cov == nil {
		cov = &models.AreaCoverage{Area: c.Area}
		coverage[c.Area] = cov
	}
	if requested {
		cov.Requested++
	}
	if c.Status != "active" {
		return
	}
	cov.Total++
	switch state {
	case learning.Completed:
		cov.Completed++
	case learning.Known:
		cov.Known++
	case learning.InProgress:
		cov.InProgress++
	}
	if dismissed {
		cov.Dismissed++
	}
}
