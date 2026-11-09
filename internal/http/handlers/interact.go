package handlers

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/wooo-jin/punisher/internal/models"
)

// ListQuestions handles GET /v1/lectures/{id}/questions (the user's thread).
func (h *LearnHandler) ListQuestions(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	lec, err := h.Store.GetLecture(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, "list questions", err)
		return
	}
	list, err := h.Store.ListQuestions(r.Context(), userID, lec.ID)
	if err != nil {
		writeServiceError(w, r, "list questions", err)
		return
	}
	if list == nil {
		list = []models.Question{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"questions": list})
}

type askBody struct {
	Anchor   string `json:"anchor"`
	Question string `json:"question"`
}

// Ask handles POST /v1/lectures/{id}/questions — answered synchronously with
// the lecture section and concept context. A failed LLM call returns 502 with
// the stored failed entry (status=failed), never a fake answer.
func (h *LearnHandler) Ask(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body askBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	body.Question = strings.TrimSpace(body.Question)
	if body.Question == "" || utf8.RuneCountInString(body.Question) > 2000 {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "question is required (max 2000 chars)")
		return
	}
	if !h.requireOwner(w, r, userID) {
		return
	}
	lec, err := h.Store.GetLecture(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, "ask", err)
		return
	}
	q, err := h.Svc.Ask(r.Context(), userID, lec, body.Anchor, body.Question)
	if err != nil {
		writeServiceError(w, r, "ask", err)
		return
	}
	status := http.StatusCreated
	if q.Status == "failed" {
		status = http.StatusBadGateway
	}
	writeJSON(w, status, map[string]any{"question": q})
}

// GetQuestion handles GET /v1/questions/{id} (poll alt explanations).
func (h *LearnHandler) GetQuestion(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	q, err := h.Store.GetQuestion(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, "get question", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"question": q})
}

// Gloss handles GET /v1/terms/{conceptId}/gloss (cached short explanation).
func (h *LearnHandler) Gloss(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	conceptID := chi.URLParam(r, "conceptId")
	gloss, cached, err := h.Svc.Gloss(r.Context(), userID, conceptID)
	if err != nil {
		writeServiceError(w, r, "gloss", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"conceptId": conceptID, "gloss": gloss, "cached": cached})
}

type noteBody struct {
	LectureID *string `json:"lectureId"`
	ConceptID *string `json:"conceptId"`
	Anchor    *string `json:"anchor"`
	Kind      *string `json:"kind"`
	Quote     *string `json:"quote"`
	Body      *string `json:"body"`
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func tooLong(limit int, values ...*string) bool {
	for _, v := range values {
		if v != nil && utf8.RuneCountInString(*v) > limit {
			return true
		}
	}
	return false
}

// ListNotes handles GET /v1/notes?lectureId=&conceptId=.
func (h *LearnHandler) ListNotes(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	notes, err := h.Store.ListNotes(r.Context(), userID, q.Get("lectureId"), q.Get("conceptId"))
	if err != nil {
		writeServiceError(w, r, "list notes", err)
		return
	}
	if notes == nil {
		notes = []models.Note{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"notes": notes})
}

// CreateNote handles POST /v1/notes.
func (h *LearnHandler) CreateNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body noteBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	kind := str(body.Kind)
	if kind == "" {
		kind = "note"
	}
	if (kind != "note" && kind != "highlight") || (str(body.LectureID) == "" && str(body.ConceptID) == "") ||
		tooLong(10000, body.Body, body.Quote) || tooLong(200, body.Anchor) {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "lectureId or conceptId required; kind note|highlight; body max 10000")
		return
	}
	n, err := h.Store.CreateNote(r.Context(), userID, models.Note{
		LectureID: str(body.LectureID), ConceptID: str(body.ConceptID), Anchor: str(body.Anchor),
		Kind: kind, Quote: str(body.Quote), Body: str(body.Body),
	})
	if err != nil {
		writeServiceError(w, r, "create note", err)
		return
	}
	writeJSON(w, http.StatusCreated, n)
}

// UpdateNote handles PATCH /v1/notes/{id}.
func (h *LearnHandler) UpdateNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body noteBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	if tooLong(10000, body.Body, body.Quote) || tooLong(200, body.Anchor) {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "body max 10000, anchor max 200")
		return
	}
	n, err := h.Store.UpdateNote(r.Context(), userID, chi.URLParam(r, "id"), body.Anchor, body.Quote, body.Body)
	if err != nil {
		writeServiceError(w, r, "update note", err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

// DeleteNote handles DELETE /v1/notes/{id}.
func (h *LearnHandler) DeleteNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	if err := h.Store.DeleteNote(r.Context(), userID, chi.URLParam(r, "id")); err != nil {
		writeServiceError(w, r, "delete note", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
