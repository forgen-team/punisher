package handlers

import (
	"errors"
	"net/http"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/services"
	"github.com/forgen-team/punisher/internal/store"
	"github.com/go-chi/chi/v5"
)

// ErrorChapterInvalid: the model's chapter design failed validation twice.
const ErrorChapterInvalid ErrorCode = "CHAPTER_INVALID"

const chapterQuestion = "다음엔 어떤 걸 공부해 보고 싶어요?"

// ChapterSuggestions handles GET /v1/chapters/suggestions.
func (h *LearnHandler) ChapterSuggestions(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	list, err := h.Svc.SuggestChapters(r.Context(), userID)
	if err != nil {
		writeServiceError(w, r, "chapter suggestions", err)
		return
	}
	if list == nil {
		list = []learning.Suggestion{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"question": chapterQuestion, "suggestions": list})
}

// CreateChapterDraft handles POST /v1/chapters/drafts {suggestionIds, freeText} —
// designs the chapter (LLM) and returns a preview; the map is unchanged.
func (h *LearnHandler) CreateChapterDraft(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body services.ChapterRequest
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	if _, err := services.ValidateChapterRequest(body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	if !h.requireOwner(w, r, userID) {
		return
	}
	res, err := h.Svc.CreateChapterDraft(r.Context(), userID, body)
	if writeChapterError(w, r, "create chapter draft", err) {
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

// ApplyChapterDraft handles POST /v1/chapters/drafts/{id}/apply ('이대로 열기').
func (h *LearnHandler) ApplyChapterDraft(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	id, err := h.Svc.ApplyChapterDraft(r.Context(), userID, chi.URLParam(r, "id"))
	if writeChapterError(w, r, "apply chapter draft", err) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"chapterId": id})
}

// ListChapters handles GET /v1/chapters.
func (h *LearnHandler) ListChapters(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	list, err := h.Store.ListChapters(r.Context(), userID)
	if err != nil {
		writeServiceError(w, r, "list chapters", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"chapters": list})
}

// writeChapterError writes chapter-specific errors; returns true when it wrote.
func writeChapterError(w http.ResponseWriter, r *http.Request, op string, err error) bool {
	if err == nil {
		return false
	}
	var design *services.ChapterDesignError
	switch {
	case errors.As(err, &design):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": map[string]any{
			"code": ErrorChapterInvalid, "message": "챕터 설계가 검증을 통과하지 못했습니다. 주제를 조금 더 구체적으로 적어 다시 시도해 주세요.",
			"problems": design.Problems,
		}})
	case errors.Is(err, services.ErrInvalidChapterRequest):
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
	case errors.Is(err, store.ErrDraftApplied):
		writeError(w, r, http.StatusConflict, ErrorConflict, "이미 연 챕터입니다")
	default:
		writeServiceError(w, r, op, err)
	}
	return true
}
