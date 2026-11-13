package handlers

import (
	"net/http"
	"strings"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/store"
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
	MasteryTarget   *int    `json:"masteryTarget"`
	ReviewIntervals []int   `json:"reviewIntervals"`
	ReviewMinutes   *int    `json:"reviewMinutes"`
	ReviewGrader    *string `json:"reviewGrader"`
	PretestEnabled  *bool   `json:"pretestEnabled"`
}

var dailyMinuteOptions = map[int]bool{15: true, 30: true, 45: true, 60: true}

// Codes of the settings 400 responses (error.code; the web shows a message per code).
const (
	ErrorInvalidLanguage      ErrorCode = "INVALID_LANGUAGE"
	ErrorInvalidGrader        ErrorCode = "INVALID_GRADER"
	ErrorInvalidDailyMinutes  ErrorCode = "INVALID_DAILY_MINUTES"
	ErrorInvalidMasteryTarget ErrorCode = "INVALID_MASTERY_TARGET"
	ErrorInvalidIntervals     ErrorCode = "INVALID_INTERVALS"
	ErrorInvalidReviewMinutes ErrorCode = "INVALID_REVIEW_MINUTES"
	ErrorSettingsRequired     ErrorCode = "SETTINGS_REQUIRED"
)

// settingsError is a rejected field: the 400 code and the (English) message.
type settingsError struct {
	code ErrorCode
	msg  string
}

// settingsPatch validates the given fields; nil error = valid.
func settingsPatch(body settingsBody) (store.LearningSettingsPatch, *settingsError) {
	p := store.LearningSettingsPatch{DailyMinutes: body.DailyMinutes, MasteryTarget: body.MasteryTarget,
		ReviewIntervals: body.ReviewIntervals, ReviewMinutes: body.ReviewMinutes, PretestEnabled: body.PretestEnabled}
	if body.ExampleLanguage != nil {
		lang := strings.ToLower(strings.TrimSpace(*body.ExampleLanguage))
		if !learning.IsValidExampleLanguage(lang) {
			return p, &settingsError{ErrorInvalidLanguage, "exampleLanguage must be one of " + strings.Join(learning.ExampleLanguageIDs, "|")}
		}
		p.ExampleLanguage = &lang
	}
	if body.ReviewGrader != nil {
		grader := strings.ToLower(strings.TrimSpace(*body.ReviewGrader))
		if !learning.ValidReviewGrader(grader) {
			return p, &settingsError{ErrorInvalidGrader, "reviewGrader must be self|llm"}
		}
		p.ReviewGrader = &grader
	}
	switch {
	case body.DailyMinutes != nil && !dailyMinuteOptions[*body.DailyMinutes]:
		return p, &settingsError{ErrorInvalidDailyMinutes, "dailyMinutes must be 15|30|45|60"}
	case body.MasteryTarget != nil && !learning.ValidMasteryTarget(*body.MasteryTarget):
		return p, &settingsError{ErrorInvalidMasteryTarget, "masteryTarget must be 3~5"}
	case body.ReviewIntervals != nil && !learning.ValidReviewIntervals(body.ReviewIntervals):
		return p, &settingsError{ErrorInvalidIntervals, "reviewIntervals must be 1~6 increasing positive day counts"}
	case body.ReviewMinutes != nil && !learning.ValidReviewMinutes(*body.ReviewMinutes):
		return p, &settingsError{ErrorInvalidReviewMinutes, "reviewMinutes must be 0~30"}
	case p.ExampleLanguage == nil && p.DailyMinutes == nil && p.MasteryTarget == nil && p.ReviewIntervals == nil &&
		p.ReviewMinutes == nil && p.ReviewGrader == nil && p.PretestEnabled == nil:
		return p, &settingsError{ErrorSettingsRequired, "at least one setting is required"}
	}
	return p, nil
}

// PutSettings handles PUT /v1/me/settings (partial: only the given fields change).
// Affects lectures/answers generated afterwards; existing lectures are not regenerated.
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
	patch, invalid := settingsPatch(body)
	if invalid != nil {
		writeError(w, r, http.StatusBadRequest, invalid.code, invalid.msg)
		return
	}
	st, err := h.Store.UpdateLearningSettings(r.Context(), userID, patch)
	if err != nil {
		writeServiceError(w, r, "save settings", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": st, "exampleLanguages": languageOptions()})
}
