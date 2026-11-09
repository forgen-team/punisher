package handlers

import (
	"net/http"
	"strings"

	"github.com/wooo-jin/punisher/internal/learning"
)

type languageOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func languageOptions() []languageOption {
	out := make([]languageOption, 0, len(learning.ExampleLanguageIDs))
	for _, id := range learning.ExampleLanguageIDs {
		out = append(out, languageOption{ID: id, Label: learning.ExampleLanguages[id]})
	}
	return out
}

// GetSettings handles GET /v1/me/settings (defaults when never saved).
func (h *LearnHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	st, err := h.Store.GetLearningSettings(r.Context(), userID)
	if err != nil {
		writeServiceError(w, r, "get settings", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": st, "exampleLanguages": languageOptions()})
}

type settingsBody struct {
	ExampleLanguage *string `json:"exampleLanguage"`
	DailyMinutes    *int    `json:"dailyMinutes"`
}

var dailyMinuteOptions = map[int]bool{15: true, 30: true, 45: true, 60: true}

// PutSettings handles PUT /v1/me/settings. Affects lectures/answers generated
// afterwards; existing lectures are not regenerated.
func (h *LearnHandler) PutSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body settingsBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	lang, daily := "", 0
	if body.ExampleLanguage != nil {
		lang = strings.ToLower(strings.TrimSpace(*body.ExampleLanguage))
		if !learning.IsValidExampleLanguage(lang) {
			writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "exampleLanguage must be one of "+strings.Join(learning.ExampleLanguageIDs, "|"))
			return
		}
	}
	if body.DailyMinutes != nil {
		if !dailyMinuteOptions[*body.DailyMinutes] {
			writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "dailyMinutes must be 15|30|45|60")
			return
		}
		daily = *body.DailyMinutes
	}
	if lang == "" && daily == 0 {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "exampleLanguage or dailyMinutes is required")
		return
	}
	st, err := h.Store.SaveLearningSettings(r.Context(), userID, lang, daily)
	if err != nil {
		writeServiceError(w, r, "save settings", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": st, "exampleLanguages": languageOptions()})
}
