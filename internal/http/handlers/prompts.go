package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/wooo-jin/punisher/internal/prompts"
	"github.com/wooo-jin/punisher/internal/services"
)

// maxPromptRunes bounds a saved block body.
const maxPromptRunes = 20000

// promptBlockView is a registry block with the caller's effective body.
type promptBlockView struct {
	prompts.Block
	Body       string     `json:"body"`
	Overridden bool       `json:"overridden"`
	UpdatedAt  *time.Time `json:"updatedAt,omitempty"`
}

var feedbackKinds = []string{"too_easy", "too_hard", "insufficient", "want_more"}

// ListPrompts handles GET /v1/me/prompts — blocks in pipeline order, the
// caller's effective bodies, and the feedback → style-hint sentences (used to
// prefill an edit from lecture feedback).
func (h *LearnHandler) ListPrompts(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	overrides, err := h.Store.PromptOverrides(r.Context(), userID)
	if err != nil {
		writeServiceError(w, r, "list prompts", err)
		return
	}
	blocks := make([]promptBlockView, 0, len(prompts.Blocks))
	for _, b := range prompts.Blocks {
		v := promptBlockView{Block: b, Body: b.Default}
		if o, found := overrides[b.ID]; found {
			v.Body, v.Overridden, v.UpdatedAt = o.Body, true, &o.UpdatedAt
		}
		blocks = append(blocks, v)
	}
	hints := map[string]string{}
	for _, k := range feedbackKinds {
		hints[k] = prompts.StyleHintFor(k)
	}
	writeJSON(w, http.StatusOK, map[string]any{"blocks": blocks, "feedbackHints": hints, "version": prompts.Version})
}

type promptBody struct {
	Body      *string `json:"body"`
	HistoryID int64   `json:"historyId"`
}

// validatePromptBody trims trailing whitespace and bounds the size.
func validatePromptBody(body string) (string, error) {
	body = strings.TrimRight(body, " \t\r\n")
	if strings.TrimSpace(body) == "" {
		return "", errors.New("body is required (use DELETE to restore the default)")
	}
	if utf8.RuneCountInString(body) > maxPromptRunes {
		return "", errors.New("body max 20000 chars")
	}
	return body, nil
}

func promptBlock(w http.ResponseWriter, r *http.Request) (prompts.Block, bool) {
	b, ok := prompts.LookupBlock(chi.URLParam(r, "id"))
	if !ok {
		writeError(w, r, http.StatusNotFound, ErrorNotFound, "unknown prompt block")
	}
	return b, ok
}

// SavePrompt handles PUT /v1/me/prompts/{id} {body}. Saving the default text
// removes the override so the recorded version stays the code version.
func (h *LearnHandler) SavePrompt(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	block, ok := promptBlock(w, r)
	if !ok {
		return
	}
	var req promptBody
	if err := decodeJSON(r, &req); err != nil || req.Body == nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "body is required")
		return
	}
	body, err := validatePromptBody(*req.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	h.storePrompt(w, r, userID, block, body, "save")
}

func (h *LearnHandler) storePrompt(w http.ResponseWriter, r *http.Request, userID string, block prompts.Block, body, action string) {
	if isDefaultBody(block, body) {
		if err := h.Store.ResetPromptOverride(r.Context(), userID, block.ID); err != nil {
			writeServiceError(w, r, "reset prompt", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"block": promptBlockView{Block: block, Body: block.Default}, "unknownVars": []string{}})
		return
	}
	o, err := h.Store.SavePromptOverride(r.Context(), userID, block.ID, body, action)
	if err != nil {
		writeServiceError(w, r, "save prompt", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"block":       promptBlockView{Block: block, Body: o.Body, Overridden: true, UpdatedAt: &o.UpdatedAt},
		"unknownVars": prompts.UnknownVars(body, block.Vars),
	})
}

// ResetPrompt handles DELETE /v1/me/prompts/{id} — back to the code default.
func (h *LearnHandler) ResetPrompt(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	block, ok := promptBlock(w, r)
	if !ok {
		return
	}
	if err := h.Store.ResetPromptOverride(r.Context(), userID, block.ID); err != nil {
		writeServiceError(w, r, "reset prompt", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"block": promptBlockView{Block: block, Body: block.Default}})
}

// PromptHistory handles GET /v1/me/prompts/{id}/history (newest first, max 20).
func (h *LearnHandler) PromptHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	block, ok := promptBlock(w, r)
	if !ok {
		return
	}
	list, err := h.Store.PromptHistory(r.Context(), userID, block.ID)
	if err != nil {
		writeServiceError(w, r, "prompt history", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": list})
}

// RestorePrompt handles POST /v1/me/prompts/{id}/restore {historyId}.
func (h *LearnHandler) RestorePrompt(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	block, ok := promptBlock(w, r)
	if !ok {
		return
	}
	var req promptBody
	if err := decodeJSON(r, &req); err != nil || req.HistoryID <= 0 {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "historyId is required")
		return
	}
	entry, err := h.Store.PromptHistoryEntryByID(r.Context(), userID, req.HistoryID)
	if err != nil {
		writeServiceError(w, r, "restore prompt", err)
		return
	}
	if entry.PromptID != block.ID {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "history entry belongs to another block")
		return
	}
	h.storePrompt(w, r, userID, block, entry.Body, "restore")
}

// PreviewPrompt handles POST /v1/me/prompts/{id}/preview {body?} — the fully
// assembled prompt with an example concept; no LLM call.
func (h *LearnHandler) PreviewPrompt(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var req promptBody
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
			return
		}
	}
	if req.Body != nil && utf8.RuneCountInString(*req.Body) > maxPromptRunes {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "body max 20000 chars")
		return
	}
	p, err := h.Svc.PreviewPrompt(r.Context(), userID, chi.URLParam(r, "id"), req.Body)
	if errors.Is(err, services.ErrUnknownPrompt) {
		writeError(w, r, http.StatusNotFound, ErrorNotFound, err.Error())
		return
	}
	if err != nil {
		writeServiceError(w, r, "preview prompt", err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// Regenerate handles POST /v1/lectures/{id}/regenerate — new version of the
// same target with the caller's current prompts (202 + stream URL).
func (h *LearnHandler) Regenerate(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	if !h.requireOwner(w, r, userID) {
		return
	}
	resp, err := h.Svc.RegenerateLecture(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, "regenerate lecture", err)
		return
	}
	writeJSON(w, http.StatusAccepted, resp)
}

// isDefaultBody reports whether body equals the block default (ignoring
// trailing whitespace) — saving it means "no override".
func isDefaultBody(block prompts.Block, body string) bool {
	return strings.TrimRight(body, " \t\r\n") == strings.TrimRight(block.Default, " \t\r\n")
}
